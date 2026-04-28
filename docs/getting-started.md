# Getting Started Guide

This guide walks you through setting up Ancient Coins for the first time, adding your first coin, and using the import/export features to manage your collection data.

## First-Time Setup

### Required Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | **Yes** (production) | dev default | JWT signing key. Must be 32+ characters. In release mode (`GIN_MODE=release`), the app fatals on startup if missing. |
| `WEBAUTHN_RP_ID` | No | `localhost` | Domain name for WebAuthn Relying Party ID |
| `WEBAUTHN_ORIGIN` | No | `http://localhost:8080` | Full origin URL for WebAuthn (comma-separated for multiple) |
| `CORS_ORIGINS` | No | *(smart defaults)* | Comma-separated allowed CORS origins. Falls back to WebAuthn origins + localhost |
| `AGENT_SERVICE_URL` | No | `http://localhost:8081` | URL of the Python agent service |

Run `task --list` to see all available Taskfile commands.

### 1. Start the Application

Run the app using one of the methods described in the [README](../README.md):

```sh
task run                    # development (Go + Vite)
docker compose up           # production (two containers)
```

### 2. Create Your Admin Account

Navigate to `http://localhost:5173` (development) or `http://localhost:8080` (Docker).

Click **Register** and create your account. The **first user** to register is automatically assigned the **admin** role. All subsequent users will be regular users.

### 3. Configure Settings (Admin)

As admin, click **Admin** in the navigation bar to configure:

- **AI Configuration** — Select your AI Provider: **Anthropic** (recommended) for Claude models with built-in web search, or **Ollama** for self-hosted models. Configure the provider-specific settings (API key, model, URL) and optionally customize analysis prompts. The Ollama URL is configured here (default `http://localhost:11434`).
- **System** — Set the log level (`trace`, `debug`, `info`, `warn`, `error`) and configure the Numista API key for catalog lookups.
- **Logs** — View real-time application logs with level filtering and auto-refresh

### 4. Start Adding Coins

Click **➕ Add Coin** from the collection page. Fill in as many fields as you like — only **Name** is required. Toggle **"Add to wishlist"** at the bottom if you don't own the coin yet.

After saving, you can upload images (obverse, reverse, detail) from the coin detail page.

---

## Adding Coins

### Required Fields

| Field | Description |
| ----- | ----------- |
| **Name** | A descriptive name (e.g., "Augustus Denarius - Capricorn Reverse") |

### Optional Fields

| Field | Description | Example |
| ----- | ----------- | ------- |
| Category | Roman, Greek, Byzantine, Modern, Other | `Roman` |
| Material | Gold, Silver, Bronze, Copper, Electrum, Other | `Silver` |
| Denomination | Coin type | `Denarius` |
| Ruler / Emperor | Issuing authority | `Augustus` |
| Era / Date | Time period | `27 BC – 14 AD` |
| Mint | Mint location | `Rome` |
| Weight (grams) | Coin weight | `3.82` |
| Diameter (mm) | Coin diameter | `19.5` |
| Grade | Condition grade | `VF`, `EF`, `MS-65` |
| Rarity Rating | Reference catalog number | `RIC 207` |
| Obverse Inscription | Legend on the front | `CAESAR AVGVSTVS DIVI F PATER PATRIAE` |
| Reverse Inscription | Legend on the back | `C L CAESARES AVGVSTI F COS DESIG PRINC IVVENT` |
| Obverse Description | Design description | `Laureate head right` |
| Reverse Description | Design description | `Gaius and Lucius standing facing` |
| Purchase Price | Amount paid in USD | `450.00` |
| Current Value | Estimated value in USD | `600.00` |
| Purchase Date | Date acquired | `2024-03-15` |
| Store | Dealer or auction | `Heritage Auctions` |
| Reference URL | External link | `https://www.acsearch.info/...` |
| Reference Text | Link display label | `ACSearch Listing` |
| Notes | Free-text notes (Markdown supported) | Any additional info |
| Wishlist | Toggle if not yet owned | `true` / `false` |

### Uploading Images

From the coin detail page, click **Upload Image** and select:

- **Image Type** — `obverse`, `reverse`, `detail`, or `other`
- **Primary** — Check this to make the image the cover photo in the gallery view

Multiple images can be uploaded per coin. The primary obverse image is shown in the collection gallery by default.

---

## Import & Export

The import/export feature lets you back up your collection or migrate data between instances.

### Exporting Your Collection

1. Navigate to **Settings** (click the gear icon in the nav bar)
2. Under **Import / Export**, click **Export Collection**
3. Your browser will download a JSON file containing all your coins

The export includes every field for each coin in your collection. Image files are **not** included in the export — only the image metadata (file paths, types).

### Import File Format

The import endpoint accepts a **JSON array of coin objects**. Each object follows the same schema as the export. At minimum, each coin needs a `name` field. All other fields are optional.

#### Minimal Example

```json
[
  {
    "name": "Augustus Denarius"
  },
  {
    "name": "Nero Sestertius",
    "category": "Roman",
    "material": "Bronze"
  }
]
```

#### Full Example

```json
[
  {
    "name": "Augustus Denarius - Capricorn Reverse",
    "category": "Roman",
    "denomination": "Denarius",
    "ruler": "Augustus",
    "era": "27 BC – 14 AD",
    "mint": "Rome",
    "material": "Silver",
    "weightGrams": 3.82,
    "diameterMm": 19.5,
    "grade": "VF",
    "obverseInscription": "CAESAR AVGVSTVS DIVI F PATER PATRIAE",
    "reverseInscription": "C L CAESARES",
    "obverseDescription": "Laureate head of Augustus right",
    "reverseDescription": "Capricorn right, holding globe",
    "rarityRating": "RIC 207",
    "purchasePrice": 450.00,
    "currentValue": 600.00,
    "purchaseDate": "2024-03-15T00:00:00Z",
    "purchaseLocation": "Heritage Auctions",
    "notes": "Excellent toning. Ex. Smith Collection.",
    "referenceUrl": "https://www.acsearch.info/search.html?id=12345",
    "referenceText": "ACSearch Listing",
    "isWishlist": false
  }
]
```

### Field Reference for Import

| JSON Field | Type | Notes |
| ---------- | ---- | ----- |
| `name` | string | **Required**. Coin name/title. |
| `category` | string | One of: `Roman`, `Greek`, `Byzantine`, `Modern`, `Other`. Defaults to `Other`. |
| `material` | string | One of: `Gold`, `Silver`, `Bronze`, `Copper`, `Electrum`, `Other`. Defaults to `Other`. |
| `denomination` | string | Free text. |
| `ruler` | string | Ruler, emperor, or issuing authority. |
| `era` | string | Date or period (free text, e.g., `"44 BC"`, `"27 BC – 14 AD"`). |
| `mint` | string | Mint location. |
| `weightGrams` | number or null | Weight in grams. |
| `diameterMm` | number or null | Diameter in millimeters. |
| `grade` | string | Condition grade (e.g., `VF`, `EF`, `AU`, `MS-65`). |
| `obverseInscription` | string | Obverse legend text. |
| `reverseInscription` | string | Reverse legend text. |
| `obverseDescription` | string | Obverse design description. |
| `reverseDescription` | string | Reverse design description. |
| `rarityRating` | string | Catalog reference (e.g., `RIC 207`, `Sear 1625`). |
| `purchasePrice` | number or null | Purchase price in USD. |
| `currentValue` | number or null | Estimated current value in USD. |
| `purchaseDate` | string or null | ISO 8601 date (`"2024-03-15T00:00:00Z"` or `"2024-03-15"`). |
| `purchaseLocation` | string | Dealer, auction house, or seller. |
| `notes` | string | Free-text notes. Supports Markdown. |
| `aiAnalysis` | string | AI-generated analysis (typically set by the app, but can be imported). |
| `referenceUrl` | string | External URL. |
| `referenceText` | string | Display text for the reference link. |
| `isWishlist` | boolean | `true` for wishlist items, `false` (default) for owned coins. |

### Import Behavior

- The `id`, `userId`, `createdAt`, and `updatedAt` fields are **ignored** on import — new IDs are assigned and the coin is associated with your account.
- The `images` array is **ignored** — images must be uploaded separately after import.
- Each coin is imported independently. If one coin fails validation, the others still import.
- The response indicates how many coins were successfully imported:
  ```json
  { "message": "Import complete", "imported": 42 }
  ```
- **Duplicate detection is not performed** — importing the same file twice will create duplicate entries.

### Importing via the UI

1. Navigate to **Settings**
2. Under **Import / Export**, click **Choose File** and select your JSON file
3. Click **Import Collection**
4. A success message will show how many coins were imported

### Importing via the API

You can also import directly via the REST API:

```sh
curl -X POST http://localhost:8080/api/user/import \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d @my-coins.json
```

---

## AI Coin Analysis

Upload photos of a coin and click **Analyze with AI** on the coin detail page. The request is routed through the Python agent service using whichever AI provider is configured (Anthropic or Ollama). The AI will analyze the images and return a Markdown-formatted report covering:

- Coin identification (ruler, denomination, mint)
- Obverse and reverse design descriptions
- Inscription readings
- Condition assessment
- Historical context
- Estimated market value range

### Setup

1. Install [Ollama](https://ollama.ai/)
2. Pull a vision model:
   ```sh
   ollama pull llava
   ```
3. Start Ollama:
   ```sh
   ollama serve
   ```
4. In the app, go to **Admin → AI Configuration** and set the Ollama URL to point to your Ollama instance (default: `http://localhost:11434`)

### Custom Prompts

Admins can customize the AI analysis prompt in **Admin → AI Configuration → Analysis Prompt**. Leave blank to use the built-in numismatic analysis prompt. A custom prompt receives the coin images and should instruct the model on what analysis to perform.
