package goway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Crear un logger con logrus
		logger := logrus.New()

		// Configurar el formato del logger
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})

		// Registrar la solicitud recibida
		logger.Infof("Received request: %s %s", r.Method, r.URL.Path)

		// Medir el tiempo de ejecución de la solicitud
		start := time.Now()

		// Llamar al siguiente handler
		next.ServeHTTP(w, r)

		// Registrar el tiempo que tomó la solicitud
		logger.Infof("Request %s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Definición del tipo de manejador
type GoWayHandlerFunc func(h *GoWayContext)

// GoWay framework
type GoWay struct {
	routes map[string]GoWayHandlerFunc
}

// Constructor
func NewGoWay() *GoWay {
	return &GoWay{
		routes: make(map[string]GoWayHandlerFunc),
	}
}

// Método para ejecutar el servidor
func (g *GoWay) Run(addr string, ctx context.Context) error {
	mux := http.NewServeMux()

	mux.Handle("/", LoggerMiddleware(mux))
	for pattern, handler := range g.routes {
		logrus.Infof("Registered route: %s", pattern) // Log de la ruta registrada
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			ctx := NewGoWayContext(w, r)
			handler(ctx)
		})
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Ejecutar el servidor en una goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server shut down", err)
		}
	}()

	// Esperar la señal de terminación
	<-ctx.Done()
	// Crear contexto con timeout para apagar el servidor
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Shutting down server...")

	return srv.Shutdown(ctxShutDown)
}

// Registrar rutas
func (g *GoWay) Handle(method, pattern string, handler GoWayHandlerFunc) {
	g.routes[fmt.Sprintf("%s %s", method, pattern)] = handler
}

func (g *GoWay) GET(pattern string, handler GoWayHandlerFunc) {
	g.Handle("GET", pattern, handler)
}

func (g *GoWay) POST(pattern string, handler GoWayHandlerFunc) {
	g.Handle("POST", pattern, handler)
}

// GoWayContext maneja la petición y respuesta
type GoWayContext struct {
	w http.ResponseWriter
	r *http.Request
}

// Constructor del contexto
func NewGoWayContext(w http.ResponseWriter, r *http.Request) *GoWayContext {
	return &GoWayContext{w, r}
}

// Obtener parámetro de query
func (c *GoWayContext) QueryParam(key string) string {
	return c.r.URL.Query().Get(key)
}

// Leer JSON del cuerpo de la petición
func (c *GoWayContext) Body(v interface{}) error {
	body, err := io.ReadAll(c.r.Body)
	if err != nil {
		return err
	}
	defer c.r.Body.Close()
	return json.Unmarshal(body, v)
}

// Enviar respuesta JSON
func (c *GoWayContext) JSON(status int, data interface{}) {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(status)
	json.NewEncoder(c.w).Encode(data)
}

// Obtener un valor del header (simulación de middleware)
func (c *GoWayContext) GetString(header string) string {
	return c.r.Header.Get(header)
}
