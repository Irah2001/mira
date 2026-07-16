package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Config stocke la configuration du serveur MCP
type Config struct {
	BaseURL string
	Client  *http.Client
}

// APIResponse enveloppe la structure stable de Mira
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error,omitempty"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	apiURL := os.Getenv("MIRA_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080/api/v1"
	}

	cfg := &Config{
		BaseURL: apiURL,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}

	slog.Info("Démarrage du serveur MCP Mira", "api_url", cfg.BaseURL)

	// Initialisation du serveur MCP
	s := server.NewMCPServer(
		"mira-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// ---------------------------------------------------------
	// 1. TOOL : search_notes
	// ---------------------------------------------------------
	searchTool := mcp.NewTool("search_notes",
		mcp.WithDescription("Recherche hybride (full-text + sémantique) dans les notes Mira"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Le texte ou mot-clé à rechercher"),
		),
	)
	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Format d'arguments invalide"), nil
		}

		query, ok := args["query"].(string)
		if !ok {
			return mcp.NewToolResultError("Le paramètre 'query' est manquant ou invalide"), nil
		}

		url := fmt.Sprintf("%s/search?q=%s", cfg.BaseURL, query)
		return cfg.forwardGetRequest(ctx, url)
	})

	// ---------------------------------------------------------
	// 2. TOOL : get_note
	// ---------------------------------------------------------
	getNoteTool := mcp.NewTool("get_note",
		mcp.WithDescription("Récupère les détails complets d'une note spécifique par son ID (contenu, résumé, tags, score)"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("L'identifiant unique numérique de la note"),
		),
	)
	s.AddTool(getNoteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Format d'arguments invalide"), nil
		}

		idStr, ok := args["id"].(string)
		if !ok {
			return mcp.NewToolResultError("Le paramètre 'id' est manquant ou invalide"), nil
		}

		url := fmt.Sprintf("%s/notes/%s", cfg.BaseURL, idStr)
		return cfg.forwardGetRequest(ctx, url)
	})

	// ---------------------------------------------------------
	// 3. TOOL : list_recent_notes
	// ---------------------------------------------------------
	listTool := mcp.NewTool("list_recent_notes",
		mcp.WithDescription("Liste les dernières notes créées ou modifiées"),
		mcp.WithNumber("limit",
			mcp.Description("Nombre maximum de notes à renvoyer (défaut 10)"),
		),
	)
	s.AddTool(listTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Format d'arguments invalide"), nil
		}

		limit := 10
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}

		url := fmt.Sprintf("%s/notes?limit=%d", cfg.BaseURL, limit)
		return cfg.forwardGetRequest(ctx, url)
	})

	// ---------------------------------------------------------
	// 4. TOOL : add_note
	// ---------------------------------------------------------
	addNoteTool := mcp.NewTool("add_note",
		mcp.WithDescription("Crée une nouvelle note. Déclenche immédiatement l'enrichissement sémantique asynchrone"),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Titre explicite de la note"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Corps textuel détaillé de la note"),
		),
		mcp.WithString("tags",
			mcp.Description("Tags optionnels séparés par des virgules (ex: 'go, web, ia')"),
		),
	)
	s.AddTool(addNoteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Format d'arguments invalide"), nil
		}

		title, okTitle := args["title"].(string)
		content, okContent := args["content"].(string)

		if !okTitle || !okContent {
			return mcp.NewToolResultError("Les paramètres 'title' et 'content' sont requis"), nil
		}

		var tags []string
		if tagsStr, ok := args["tags"].(string); ok && tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				tags = append(tags, strings.TrimSpace(t))
			}
		}

		payload := map[string]interface{}{
			"title":   title,
			"content": content,
			"tags":    tags,
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/notes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := cfg.Client.Do(req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erreur lors de l'appel à l'API Mira : %v", err)), nil
		}
		defer resp.Body.Close()

		var apiResp APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return mcp.NewToolResultError("Impossible de décoder la réponse de l'API"), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Note créée avec succès ! Contenu brut : %s", string(apiResp.Data))), nil
	})

	// Lancement de l'écoute MCP sur l'entrée/sortie standard (stdio)
	if err := server.ServeStdio(s); err != nil {
		slog.Error("🔴 Erreur fatale sur le canal Stdio du serveur MCP", "error", err)
		os.Exit(1)
	}
}

// Fonction utilitaire pour éviter la duplication des appels GET
func (cfg *Config) forwardGetRequest(ctx context.Context, url string) (*mcp.CallToolResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("L'API Mira ne répond pas : %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return mcp.NewToolResultError("Ressource introuvable."), nil
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Erreur de lecture du JSON de l'API."), nil
	}

	return mcp.NewToolResultText(string(apiResp.Data)), nil
}
