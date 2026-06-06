# Coin Details

> Store comprehensive numismatic data, images, references, activity logs, and notes for each coin in your collection.

## Overview

Each coin in your collection can hold rich, structured metadata that captures its numismatic significance, provenance, valuations, and personal notes. The detail page is the primary interface for viewing and editing all coin information.

## Core Fields

### Identification
- **Name** — Coin description or catalog name
- **Denomination** — Value (denarius, aureus, sestertius, etc.)
- **Ruler** — Authority or monarch on the coin
- **Era** — Time period (ancient, medieval, modern)
- **Category** — Roman, Greek, Byzantine, Modern, or Other

### Physical Characteristics
- **Material** — Gold, silver, bronze, copper, electrum, or other
- **Weight** — In grams
- **Diameter** — In millimeters
- **Thickness** — Optional thickness measurement
- **Grade** — Numerical grade (VF, EF, AU, MS, etc.)

### Numismatic Details
- **Mint Mark** — Mint location indicator
- **Obverse Inscription** — Text on the front
- **Reverse Inscription** — Text on the back
- **Structured Catalog References** — Formal references (RIC, RPC, SNG, Numista, etc.) with catalog, volume, number, optional invoice number, and authority URI

### Financial Data
- **Purchase Price** — Original acquisition cost
- **Purchase Date** — When acquired
- **Current Value** — Latest estimated value
- **Dealer/Source** — Where acquired
- **Cost Basis** — Purchase price + any restoration costs

### Provenance & History
- **Provenance Notes** — Where the coin came from and any prior ownership
- **Condition Notes** — Any cleaning, restoration, or damage
- **Activity Journal** — Timestamped entries (e.g., "sent to NGC", "displayed at show", "cleaned")
- **Free-Text Notes** — Any additional information

## Images

### Image Management
- **Upload Multiple Images** — Support for obverse, reverse, edge, detail, full views
- **Upload Methods**:
  - Direct file upload
  - Paste image URL (via server proxy fetch)
  - Camera capture in PWA mode
- **Image Gallery** — Lightbox viewer with zoom and navigation
- **Set Primary Image** — Choose which image appears on coin cards in collection view

### Image Operations
- **Background Removal** — Remove image backgrounds in-place using client-side ML
- **Circle Clipping** — Automatically clip obverse/reverse to circular framing (gated behind optional `circleClip` parameter)
- **OCR Text Extraction** — Extract text from store cards, certificates, or labels

## AI Analysis

### Analysis Workflow
1. Upload photos of obverse and reverse
2. Click **Analyze with AI**
3. Choose provider (Anthropic Claude or Ollama)
4. System analyzes each side separately with tailored prompts
5. Review analysis and accept/reject to save

### Analysis Content
Includes identification, description, inscriptions, condition assessment, historical context, and estimated market value. Stored separately for obverse and reverse.

### Configuration
- **Admin → AI Configuration** — Select provider and model
- **Custom Prompts** — Customize analysis prompts for obverse, reverse, and text extraction

## Structured Catalog References

### Reference Management
- **Add Multiple References** — Track coins across different catalogs (RIC, RPC, SNG, Numista, etc.)
- **Reference Fields**:
  - Catalog (required): which reference catalog
  - Volume (required for some catalogs)
  - Number: the catalog entry number
  - Invoice number and authority URI when available
  - Authority URI: optional link to the authoritative source

### Built-in Catalogs
- **RIC** — Roman Imperial Coinage (requires volume)
- **RPC** — Roman Provincial Coinage (requires volume)
- **SNG** — Sylloge Nummorum Greacorum (requires volume)
- **Numista** — Numista catalog ID
- **Other** — Custom catalog references

### Numista Integration
- Direct search from the coin detail page
- Find matching coins by name, denomination, and ruler
- View thumbnails and link to full Numista entries
- Auto-suggest catalog references

## Activity Journal

Track the coin's history with timestamped entries:

- **Event Types**:
  - "Acquired" (automatic on creation)
  - "Cleaned"
  - "Sent for grading" / "Received from grading"
  - "Displayed at show"
  - "Restored"
  - Custom entries

- **Add Entries** — Click the plus icon in Activity section
- **Delete Entries** — Remove entries you no longer need

## Wish List & Sold Status

- **Wishlist** — Mark coins you'd like to acquire (excludes from main collection stats)
- **Sold** — Mark coins you've sold with sale price and buyer information
- **Status Transitions** — Move coins between collection, wishlist, and sold states

## Privacy Controls

- **Private Coin** — Hide from followers and public showcases
- **Affects** — Social features and collection sharing

## History & Versioning

- **Last Updated** — Timestamp of last modification
- **Update Log** — View past changes (when available)

## Related Features

- See [AI Coin Analysis](ai-analysis.md) for vision-model analysis
- [Numista Catalog Lookup](numista-integration.md) for catalog references
- [Custom Tags](custom-tags.md) to categorize the coin
- [Coin Sets](coin-sets.md) to group coins thematically

## API Endpoints

```
GET    /api/coins/:id               # Get coin details with all preloaded associations
PUT    /api/coins/:id               # Update coin fields
POST   /api/coins/:id/analyze       # Analyze coin images
DELETE /api/coins/:id/analyze       # Delete saved AI analysis
GET    /api/coins/:id/references    # List catalog references
POST   /api/coins/:id/references    # Add catalog reference
PUT    /api/coins/:id/references/:referenceId # Update reference
DELETE /api/coins/:id/references/:referenceId # Delete reference
GET    /api/coins/:id/journal       # List journal entries
POST   /api/coins/:id/journal       # Add activity journal entry
DELETE /api/coins/:id/journal/:entryId # Delete journal entry
```

## See Also

- [Collection Management](collection-management.md) — Browse and organize coins
- [Collection Statistics](statistics.md) — Analytics across your collection
- [PDF Export](pdf-export.md) — Export collection with full coin details
