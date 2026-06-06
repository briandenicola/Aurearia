# Sold Coins

> Track coins you've sold with comprehensive profit/loss analysis and valuation history.

## Overview

The Sold Coins feature provides visibility into coins you've sold with detailed financial tracking and performance analysis.

## Features

- **Sell from Detail Page** — Click "Sell" on any coin to move it to sold status
- **Sale Information** — Capture sale price and buyer name
- **Profit/Loss Tracking** — Automatic calculation and visualization
  - Green: profit (sale price > purchase price)
  - Red: loss (sale price < purchase price)
- **Sold Gallery** — Dedicated view of all sold coins
- **Valuation History** — See what you paid vs. what you sold for
- **Stats Separation** — Sold coins excluded from active collection totals
- **Export** — Include sold coins in PDF/CSV exports with profit/loss data

## Workflow

1. Open any coin in your collection
2. Click **Sell** button
3. Enter **Sale Price** and **Buyer Name**
4. Confirm to move coin to sold status
5. View in **Sold Coins** gallery with profit/loss visible

## Configuration

- No special configuration needed
- Works with any coin in your collection

## API Endpoints

```
POST   /api/coins/:id/sell          # Mark coin as sold with price/buyer
GET    /api/coins?status=sold       # List sold coins
GET    /api/stats                  # Portfolio stats include sold coin metrics
```

See also: [Collection Management](collection-management.md), [Collection Statistics](statistics.md)
