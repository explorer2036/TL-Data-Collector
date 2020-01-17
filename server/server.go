package server

import (
	"TL-Data-Collector/config"
	"context"
	"net/http"
	"sync"

	"github.com/emicklei/go-restful"
)

// Server defines the http server
type Server struct {
	server *http.Server
}

// NewServer returns a new server
func NewServer(settings *config.Config) *Server {
	s := &Server{}
	s.server = &http.Server{
		Addr:    settings.App.Server,
		Handler: s.newContainer(),
	}
	return s
}

// User for login id and password
type User struct {
	LoginId  string
	Password string
}

// Login with the login id and password
func (s *Server) Login(request *restful.Request, response *restful.Response) {
	user := new(User)
	if err := request.ReadEntity(&user); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

// newContainer returns a restful container with routes
func (s *Server) newContainer() *restful.Container {
	container := restful.NewContainer()

	ws := new(restful.WebService)
	ws.Path("/").Doc("root").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.POST("/login").To(s.Login)) // user login routes

	container.Add(ws)

	return container
}

// Start the server
func (s *Server) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.server.ListenAndServe(); err != nil {
			return
		}
	}()
}

// Stop shutdown the server
func (s *Server) Stop() {
	s.server.Shutdown(context.Background())
}
