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

type CustomError struct {
	Message    string
	StatusCode int
}

func (e *CustomError) Error() string {
	return e.Message
}

func NewCustomError(message string, statusCode int) *CustomError {
	return &CustomError{
		Message:    message,
		StatusCode: statusCode,
	}
}

// Middleware de manejo de errores mejorado con error personalizado
func ErrorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				var customErr *CustomError
				switch e := err.(type) {
				case *CustomError:
					customErr = e
				default:
					customErr = NewCustomError("Internal Server Error", http.StatusInternalServerError)
				}

				// Loguear el error
				log.Printf("Error: %v", customErr)

				// Devolver el error al cliente
				http.Error(w, customErr.Message, customErr.StatusCode)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

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
	routes      map[string]GoWayHandlerFunc
	middlewares []func(http.Handler) http.Handler // Lista de middlewares
}

// Constructor
func NewGoWay() *GoWay {
	server := &GoWay{
		routes: make(map[string]GoWayHandlerFunc),
	}
	server.Use(LoggerMiddleware)
	server.Use(ErrorHandlingMiddleware)
	return server
}

// Método para ejecutar el servidor
func (g *GoWay) Run(addr string, ctx context.Context) error {
	mux := http.NewServeMux()

	for pattern, handler := range g.routes {
		logrus.Infof("Registered route: %s", pattern) // Log de la ruta registrada
		// Crear el manejador para la ruta actual
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Crear contexto para manejar la petición
			ctx := NewGoWayContext(w, r)
			handler(ctx)
		})

		// Aplicar la cadena de middlewares y luego el manejador de la ruta
		mux.Handle(pattern, ChainMiddlewares(g.middlewares, handler))
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

func (g *GoWay) Use(middleware func(http.Handler) http.Handler) {
	g.middlewares = append(g.middlewares, middleware)
}

func ChainMiddlewares(middlewares []func(http.Handler) http.Handler, final http.Handler) http.Handler {
	// Comienza con el manejador final y aplica cada middleware en orden inverso
	for i := len(middlewares) - 1; i >= 0; i-- {
		final = middlewares[i](final)
	}
	return final
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
