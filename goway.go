package goway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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
func (g *GoWay) Run(addr string) error {
	mux := http.NewServeMux()
	for pattern, handler := range g.routes {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			ctx := NewGoWayContext(w, r)
			handler(ctx)
		})
	}
	fmt.Println("Server running on", addr)
	return http.ListenAndServe(addr, mux)
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
