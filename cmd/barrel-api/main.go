package main

import (
	"barrel-api/auth"
	"barrel-api/config"
	"barrel-api/controller"
	"barrel-api/core"
	"barrel-api/handler"
	"barrel-api/internal/mqtt"
	"barrel-api/repository"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	Router *mux.Router
}

func NewApp(cfg *config.Config) *App {
	core.InitializeDatabase("user=" + cfg.Database.User + " password=" + cfg.Database.Password + " dbname=" + cfg.Database.Name + " sslmode=disable")

	app := &App{
		Router: mux.NewRouter(),
	}
	app.Router.Use(auth.JSONMiddleware)

	authRouter := app.Router.PathPrefix("/auth/v1").Subrouter()
	v1 := app.Router.PathPrefix("/api/v1").Subrouter()

	v1.Use(auth.AuthenticationMiddleware(repository.NewSessionRepository(core.GetDB())))

	groupRepo := repository.NewGroupRepository(core.GetDB())
	groupController := controller.NewGroupController(groupRepo)
	groupHandler := handler.NewGroupHandler(groupController)
	groupHandler.RegisterRoutes(v1)

	userRepo := repository.NewUserRepository(core.GetDB())
	prov := mqtt.NewMosqDynSec("tcp://dioge.com.br:1883", "admin", cfg.MQTTPassword)
	userController := controller.NewUserController(userRepo, groupRepo, prov)
	userHandler := handler.NewUserHandler(userController)
	userHandler.RegisterRoutes(authRouter)

	sessionRepo := repository.NewSessionRepository(core.GetDB())
	sessionController := controller.NewSessionController(sessionRepo)
	sessionHandler := handler.NewSessionHandler(sessionController)
	sessionHandler.RegisterRoutes(authRouter)

	smartDeviceRepo := repository.NewSmartDeviceRepository(core.GetDB())
	smartDeviceController := controller.NewSmartDeviceController(smartDeviceRepo, groupRepo)
	smartDeviceHandler := handler.NewSmartDeviceHandler(smartDeviceController)
	smartDeviceHandler.RegisterRoutes(v1)

	deviceShareRepo := repository.NewDeviceShareRepository(core.GetDB())
	deviceShareController := controller.NewDeviceShareController(deviceShareRepo, smartDeviceRepo, groupRepo, userRepo, prov)
	deviceShareHandler := handler.NewDeviceShareHandler(deviceShareController)
	deviceShareHandler.RegisterRoutes(v1)

	return app
}

func main() {
	cfg := config.LoadConfig()

	app := NewApp(cfg)

	addr := cfg.ServerAddress
	// CORS options
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})

	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, handlers.CORS(originsOk, headersOk, methodsOk)(app.Router)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
