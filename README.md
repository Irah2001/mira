# Mira — Outil de Mémoire Personnelle (Version 3.0)

Mira est une base de connaissances locale et centralisée qui vous permet de stocker, lister, modifier et rechercher vos notes ou mémos personnels.

Cette version 3 (V3) marque une évolution architecturale majeure. Le stockage sur fichier a été remplacé par une base de données PostgreSQL sécurisée et conteneurisée via Docker. L'application introduit un pipeline d'enrichissement asynchrone ultra-rapide (Goroutines & Channels) et l'interface CLI a été refactorisée pour devenir un pur client HTTP garantissant le respect des règles métier de l'API.

---

## Architecture & Organisation du Dépôt

Le projet est structuré selon les standards modernes de l'écosystème Go :

```text
mira/
├── cmd/
│   ├── cli/
│   │   └── main.go                 # Client HTTP en ligne de commande interrogeant l'API
│   └── api/
│       └── main.go                 # Serveur API REST connecté à PostgreSQL
├── internal/
│   ├── notes/
│   │   ├── note.go                 # Modèles de domaine (avec données d'enrichissement)
│   │   ├── store.go                # Contrat/Interface NoteStore
│   │   ├── postgres.go             # Implémentation PostgreSQL (pgxpool)
│   │   └── enrich.go               # Pipeline asynchrone (Worker pool & Channels)
│   └── http/
│       └── handlers/
│           ├── middleware.go       # Middlewares (Request ID, Slog, Recovery, Timeout)
│           ├── notes.go            # Contrôleurs / Gestionnaires d'endpoints
│           ├── notes_test.go       # Tests unitaires avec store factice
│           └── response.go         # Structure de l'enveloppe JSON standardisée
├── migrations/
│   ├── 001_init.sql                # Script de création des tables relationnelles
│   └── 002_enrichment.sql          # Ajout des champs de métadonnées d'enrichissement
├── docker-compose.yml              # Configuration de la base de données (pgvector)
├── .env.example                    # Exemple de variables d'environnement (Identifiants BDD, Port)
├── go.mod
└── README.md
```

## Installation & Démarrage (Infrastructure)

Le stockage s'appuie désormais sur PostgreSQL géré via Docker Compose. Les identifiants ne sont plus écrits en dur mais gérés de manière sécurisée via le fichier .env.

1. Configurez vos variables d'environnement dans un fichier `.env` à la racine du projet (vous pouvez vous baser sur `.env.example`).

2. Démarrez la base de données PostgreSQL avec Docker Compose :
   ```bash
   docker compose up -d
   ```

3. Vérifiez que la base de données est opérationnelle :
   ```bash
   docker compose logs -f db
   ```

4. Lancez le serveur API :
   ```bash
   go run cmd/api/main.go
   ```

## Enrichissement Automatique (Asynchrone)

Afin de ne pas ralentir les requêtes de création de notes, Mira intègre un pipeline de traitement asynchrone.

- **Déclenchement au fil de l'eau** : À chaque POST ou PATCH, l'API insère la note très rapidement avec le statut `pending` et répond au client instantanément. Un job est posté dans une file d'attente (Channel interne).

- **Pool de Workers** : Des Goroutines en arrière-plan consomment ces tâches pour simuler un enrichissement métier (génération d'un résumé, calcul d'un score de pertinence, auto-découverte de tags).

- **Sécurité & Context** : Chaque tâche dispose d'un timeout strict. Si l'enrichissement échoue ou prend trop de temps, le statut en base passe à `failed`. S'il réussit, il passe à `done` et la base est mise à jour avec les nouvelles données (`summary`, `score`, `tags`).


## Utilisation de l'Interface CLI (Locale)

L'accès en ligne de commande permet des interactions rapides en local sur votre machine.

1. Ajouter une note :
   ```bash
   ./mira add "Titre de la note" "Contenu de la note"
   ```

2. Lister toutes les notes :
   ```bash
    ./mira list
    ```

3. Rechercher des notes par mot-clé :
    ```bash
    ./mira search "mot-clé"
    ```

## Utilisation de l'API HTTP REST

Le serveur API s'exécute par défaut sur le port :8080. Pour le lancer, utilisez la commande suivante :

```bash
go run cmd/api/main.go
```

### Enveloppe de Réponse JSON Stable
Afin de garantir la stabilité des clients (Web, Mobile, Extension), toutes les réponses HTTP sans exception adoptent une structure JSON unifiée :

```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

### Référence des Endpoints & Exemples de Requêtes:

| Méthode | Endpoint | Description | Exemple curl |
|---------|----------|-------------|---------------|
| GET     | api/v1/notes   | Récupère toutes les notes | `curl http://localhost:8080/api/v1/notes` |
| GET     | api/v1/notes/{id} | Récupère une note spécifique (avec tags et résumé) | `curl http://localhost:8080/api/v1/notes/1` |
| POST    | api/v1/notes   | Crée une note (démarre l'enrichissement en fond) | `curl -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" -d '{"title":"Nouvelle Note", "content":"Contenu", "tags":["web"]}'` |
| PUT     | api/v1/notes/{id} | Met à jour une note spécifique par ID | `curl -X PUT http://localhost:8080/api/v1/notes/1 -H "Content-Type: application/json" -d '{"title":"Titre mis à jour","content":"Contenu mis à jour"}'` |
| DELETE  | api/v1/notes/{id} | Supprime une note spécifique par ID | `curl -X DELETE http://localhost:8080/api/v1/notes/1` |
| GET     | api/v1/search | Recherche des notes par mot-clé | `curl http://localhost:8080/api/v1/search\?q=modules` |

### Documentation OpenAPI/Swagger

La documentation OpenAPI est disponible à l'adresse suivante : [http://localhost:8080/docs/](http://localhost:8080/docs/)
