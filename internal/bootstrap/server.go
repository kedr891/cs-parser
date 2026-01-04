package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	skins_service_api "github.com/kedr891/cs-parser/internal/api/skins_service_api"
	"github.com/kedr891/cs-parser/internal/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
	pbswagger "github.com/kedr891/cs-parser/internal/pb/swagger"
	"github.com/kedr891/cs-parser/internal/storage/pgstorage"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var globalStorage *pgstorage.Storage

type ConsumerRunner interface {
	Consume(ctx context.Context) error
}

func AppRun(
	api skins_service_api.SkinsServiceAPI,
	priceUpdateConsumer ConsumerRunner,
	storage *pgstorage.Storage,
	closeCache func(),
	log *slog.Logger,
) {
	// Сохраняем storage для REST endpoints
	globalStorage = storage
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := priceUpdateConsumer.Consume(ctx); err != nil && err != context.Canceled {
			log.Error("Price consumer failed", "error", err)
		}
	}()

	go func() {
		if err := runGRPCServer(api, log); err != nil {
			panic(fmt.Errorf("failed to run gRPC server: %v", err))
		}
	}()

	go func() {
		if err := runGatewayServer(log); err != nil {
			panic(fmt.Errorf("failed to run gateway server: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down services...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()

	storage.Close()
	closeCache()
	log.Info("Services stopped")
}

func runGRPCServer(api skins_service_api.SkinsServiceAPI, log *slog.Logger) error {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	skins_api.RegisterSkinsServiceServer(s, &api)

	log.Info("gRPC server listening on :50051")
	return s.Serve(lis)
}

func runGatewayServer(log *slog.Logger) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	swaggerPath := os.Getenv("swaggerPath")
	if swaggerPath != "" {
		if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
			log.Warn("Swagger file not found, fallback to embedded spec", "path", swaggerPath)
			swaggerPath = ""
		}
	}

	r := chi.NewRouter()

	r.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		if swaggerPath != "" {
			http.ServeFile(w, r, swaggerPath)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pbswagger.Skins())
	})

	swaggerHandler := httpSwagger.Handler(httpSwagger.URL("/swagger.json"))
	r.Get("/docs/*", swaggerHandler)

	// Backward/compatibility routes: map any /swagger* requests to the working /docs UI
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", func(w http.ResponseWriter, r *http.Request) {
		target := "/docs" + strings.TrimPrefix(r.URL.Path, "/swagger")
		if target == "/docs" || target == "/docs/" {
			target = "/docs/index.html"
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := skins_api.RegisterSkinsServiceHandlerFromEndpoint(ctx, mux, ":50051", opts)
	if err != nil {
		return err
	}

	// REST endpoint для создания скина
	r.Post("/api/v1/skins", handleCreateSkin)

	r.Mount("/", mux)

	log.Info("gRPC-Gateway server listening on :8080")
	return http.ListenAndServe(":8080", r)
}

// handleCreateSkin - REST endpoint для создания скина
func handleCreateSkin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MarketHashName string  `json:"market_hash_name"` // Опционально, генерируется автоматически
		Name           string  `json:"name"`             // Обязательно
		Weapon         string  `json:"weapon"`           // Обязательно
		Quality        string  `json:"quality"`          // Обязательно
		Rarity         string  `json:"rarity"`
		CurrentPrice   float64 `json:"current_price"`
		Currency       string  `json:"currency"`
		ImageURL       string  `json:"image_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if req.Name == "" || req.Weapon == "" || req.Quality == "" {
		http.Error(w, "Missing required fields: name, weapon, quality", http.StatusBadRequest)
		return
	}

	// Создаем новый скин (marketHashName генерируется автоматически, если не указан)
	skin := models.NewSkin(
		req.MarketHashName, // Если пустой, будет сгенерирован автоматически
		req.Name,
		req.Weapon,
		req.Quality,
	)

	skin.Rarity = req.Rarity
	skin.CurrentPrice = req.CurrentPrice
	skin.Currency = req.Currency
	if req.Currency == "" {
		skin.Currency = "USD"
	}
	skin.ImageURL = req.ImageURL

	// Сохраняем в БД
	if err := globalStorage.CreateSkin(r.Context(), skin); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create skin: %v", err), http.StatusInternalServerError)
		return
	}

	// Возвращаем результат
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Skin created successfully",
		"skin": map[string]interface{}{
			"id":               skin.ID.String(),
			"slug":             skin.Slug,
			"market_hash_name": skin.MarketHashName,
			"name":             skin.Name,
			"weapon":           skin.Weapon,
			"quality":          skin.Quality,
			"rarity":           skin.Rarity,
			"current_price":    skin.CurrentPrice,
			"currency":         skin.Currency,
			"image_url":        skin.ImageURL,
			"created_at":       skin.CreatedAt,
			"updated_at":       skin.UpdatedAt,
		},
	})
}
