<template>
  <section class="settings-section card help-section">
    <h2>Beginner's Guide to Coin Collecting</h2>
    <p class="setting-desc" style="margin-bottom: 1.5rem">
      Everything you need to know to start collecting coins with confidence.
    </p>

    <details class="help-accordion" open>
      <summary class="help-summary">Getting Started in the App</summary>
      <div class="help-content">
        <p>If this is a new account, you can open this guide from the first-login prompt or anytime from the sidebar under <strong>Getting Started</strong>.</p>

        <h4>Recommended First Steps</h4>
        <ol>
          <li>Create your account and open <strong>Settings → Data</strong>.</li>
          <li>Download the <strong>CSV Template</strong>.</li>
          <li>Fill in your coins in spreadsheet software (Excel, Google Sheets, Numbers), then export as CSV.</li>
          <li>Import the CSV from <strong>Settings → Data → Import</strong>.</li>
          <li>Review imported coins and upload photos from each coin detail page.</li>
        </ol>

        <h4>CSV Columns</h4>
        <p>Only <code>name</code> is required. All other fields are optional.</p>
        <table class="help-table">
          <thead>
            <tr><th>Column</th><th>Type</th><th>Notes</th></tr>
          </thead>
          <tbody>
            <tr><td><code>name</code></td><td>text</td><td>Required</td></tr>
            <tr><td><code>category</code></td><td>text</td><td>Roman, Greek, Byzantine, Modern, Other</td></tr>
            <tr><td><code>material</code></td><td>text</td><td>Gold, Silver, Bronze, Copper, Electrum, Other</td></tr>
            <tr><td><code>denomination</code>, <code>ruler</code>, <code>mint</code>, <code>grade</code></td><td>text</td><td>Free text fields</td></tr>
            <tr><td><code>era</code></td><td>text</td><td>Allowed values: <code>ancient</code>, <code>medieval</code>, <code>modern</code></td></tr>
            <tr><td><code>weightGrams</code>, <code>diameterMm</code>, <code>purchasePrice</code>, <code>currentValue</code></td><td>number</td><td>Use decimal values</td></tr>
            <tr><td><code>purchaseDate</code>, <code>soldDate</code></td><td>date</td><td>Use <code>YYYY-MM-DD</code></td></tr>
            <tr><td><code>isWishlist</code>, <code>isSold</code>, <code>isPrivate</code></td><td>boolean</td><td>Use true/false</td></tr>
            <tr><td><code>notes</code>, <code>referenceUrl</code>, <code>referenceText</code></td><td>text</td><td>Optional metadata (legacy reference fallback fields)</td></tr>
          </tbody>
        </table>

        <h4>CSV Example</h4>
        <pre class="help-code">name,category,material,denomination,ruler,era,mint,weightGrams,diameterMm,grade,purchasePrice,currentValue,purchaseDate,purchaseLocation,isWishlist
Augustus Denarius,Roman,Silver,Denarius,Augustus,27 BC - 14 AD,Rome,3.82,19.5,VF,450,600,2024-03-15,Heritage Auctions,false
Constantius II Follis,Roman,Bronze,Follis,Constantius II,337-361 AD,Antioch,2.90,18.1,F,35,45,2025-01-20,Local Show,false</pre>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Historical Coin Types</summary>
      <div class="help-content">
        <p>Ancient coins span thousands of years across many civilizations. Here are the major categories:</p>

        <h4>Greek (c. 600 BC – 31 BC)</h4>
        <p>Among the earliest coins ever minted. Greek coins are known for their artistic beauty and variety. Key types include:</p>
        <ul>
          <li><strong>Stater</strong> — A standard gold or silver denomination used across Greek city-states</li>
          <li><strong>Drachm / Tetradrachm</strong> — Silver coins; tetradrachms (4 drachms) are especially prized by collectors for their large size and detailed artwork</li>
          <li><strong>Obol</strong> — Small silver coin, originally 1/6 of a drachm</li>
          <li><strong>Hemidrachm</strong> — Half a drachm, commonly found in smaller denominations</li>
        </ul>
        <p class="help-tip">💡 <strong>Popular starting points:</strong> Athenian owl tetradrachms, Alexander the Great drachms, and Ptolemaic bronzes are widely available and recognizable.</p>

        <h4>Roman Republic (c. 280 – 27 BC)</h4>
        <p>Coins of the Roman Republic feature anonymous designs early on, evolving to depict political figures and events.</p>
        <ul>
          <li><strong>Denarius</strong> — The workhorse silver coin of Rome, roughly the size of a dime</li>
          <li><strong>As / Semis</strong> — Bronze denominations for everyday transactions</li>
          <li><strong>Victoriatus</strong> — Early silver coin with a Victory motif</li>
        </ul>

        <h4>Roman Imperial (27 BC – 476 AD)</h4>
        <p>The most popular category for collectors. Imperial coins always feature the emperor's portrait on the obverse.</p>
        <ul>
          <li><strong>Aureus</strong> — Gold coin (~7.7g), the most valuable denomination</li>
          <li><strong>Denarius</strong> — Silver coin, the standard unit of account</li>
          <li><strong>Sestertius</strong> — Large bronze coin, prized for detailed reverse designs</li>
          <li><strong>Antoninianus</strong> — Silver/billon coin introduced by Caracalla (215 AD), gradually debased</li>
          <li><strong>Follis</strong> — Late Roman bronze, common in the 4th century</li>
          <li><strong>Solidus</strong> — Gold coin introduced by Constantine I, replaced the aureus</li>
        </ul>
        <p class="help-tip">💡 <strong>Best for beginners:</strong> Late Roman bronzes (4th century) are affordable, plentiful, and come in recognizable emperor portraits. Start with Constantine I, Constantius II, or Valentinian I.</p>

        <h4>Byzantine (330 – 1453 AD)</h4>
        <p>Continuation of the Eastern Roman Empire. Notable for their distinctive cup-shaped coins (scyphate) and religious imagery.</p>
        <ul>
          <li><strong>Solidus / Histamenon</strong> — Gold standard coin</li>
          <li><strong>Follis</strong> — Bronze coin, often featuring Christ or the emperor with a cross</li>
          <li><strong>Hyperpyron</strong> — Later gold denomination, often cup-shaped</li>
        </ul>

        <h4>Celtic, Parthian & Other</h4>
        <p>Don't overlook less-collected areas — Celtic coins from Gaul and Britain, Parthian drachms, Sasanian silver, Judean bronzes, and Nabataean coins all offer fascinating collecting opportunities, often at lower price points.</p>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Understanding Coin Grading</summary>
      <div class="help-content">
        <p>Grading describes a coin's physical condition and is the single biggest factor in determining value. Grades are standardized across the hobby:</p>

        <table class="help-table">
          <thead>
            <tr><th>Grade</th><th>Abbreviation</th><th>Description</th></tr>
          </thead>
          <tbody>
            <tr><td>Poor</td><td>P / PO</td><td>Barely identifiable, heavy wear. Type and ruler may be unrecognizable.</td></tr>
            <tr><td>Fair</td><td>FR</td><td>Heavily worn but type is identifiable. Legend mostly gone.</td></tr>
            <tr><td>About Good</td><td>AG</td><td>Very heavily worn; outline visible, some legend readable.</td></tr>
            <tr><td>Good</td><td>G</td><td>Major design elements visible but flat. Legends partially readable.</td></tr>
            <tr><td>Very Good</td><td>VG</td><td>Main features clear with moderate wear. About half the legend readable.</td></tr>
            <tr><td>Fine</td><td>F</td><td>Moderate wear on high points. Most legend and design details visible.</td></tr>
            <tr><td>Very Fine</td><td>VF</td><td>Light wear on high points only. Full legend readable. Most detail sharp.</td></tr>
            <tr><td>Extremely Fine</td><td>EF / XF</td><td>Slight wear on highest points. Nearly full detail. Attractive coin.</td></tr>
            <tr><td>Almost Uncirculated</td><td>AU</td><td>Trace wear only. Full luster in protected areas.</td></tr>
            <tr><td>Mint State</td><td>MS</td><td>No wear at all. Varies from MS-60 (bag marks) to MS-70 (perfect).</td></tr>
            <tr><td>Superb</td><td>FDC</td><td>"Fleur de Coin" — perfect or near-perfect mint state. Extremely rare for ancients.</td></tr>
          </tbody>
        </table>

        <p class="help-tip">💡 <strong>For ancient coins</strong>, VF is considered a very respectable grade. Many ancient coins in EF or above command significant premiums. Don't expect modern-coin perfection — 2,000-year-old coins have character!</p>

        <h4>Grading Services</h4>
        <p>Professional grading services (NGC Ancients, PCGS) authenticate and grade coins, sealing them in tamper-proof holders ("slabs"). This adds cost but provides confidence in authenticity and grade. NGC Ancients is the most widely used service for ancient coins.</p>
        <p>Not all coins need to be slabbed. Many experienced collectors prefer "raw" (unslabbed) coins for their tactile appeal and lower total cost.</p>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Reference Catalogs (RIC, RPC, Sear & More)</summary>
      <div class="help-content">
        <p>Reference catalogs assign standard numbers to coin types, making identification and communication precise. When you see a number like "RIC 207," it refers to a specific coin type in a specific volume.</p>

        <table class="help-table">
          <thead>
            <tr><th>Catalog</th><th>Full Name</th><th>Coverage</th></tr>
          </thead>
          <tbody>
            <tr><td><strong>RIC</strong></td><td>Roman Imperial Coinage</td><td>The definitive reference for Roman Imperial coins (27 BC – 491 AD). 10 volumes covering every emperor. Most commonly cited catalog for Roman coins.</td></tr>
            <tr><td><strong>RRC</strong></td><td>Roman Republican Coinage</td><td>Crawford's catalog of Roman Republic coins. Referenced as "Cr." or "Crawford."</td></tr>
            <tr><td><strong>RPC</strong></td><td>Roman Provincial Coinage</td><td>Covers coins minted in Roman provinces (not Rome itself). Many volumes, also available online at rpc.ashmus.ox.ac.uk.</td></tr>
            <tr><td><strong>Sear</strong></td><td>Roman Coins and Their Values</td><td>David Sear's accessible guide, good for identification and price estimates. 5 volumes for Imperial, separate books for Greek and Byzantine.</td></tr>
            <tr><td><strong>BMC</strong></td><td>British Museum Catalogue</td><td>Scholarly reference published by the British Museum. Covers Greek, Roman, and other series.</td></tr>
            <tr><td><strong>SNG</strong></td><td>Sylloge Nummorum Graecorum</td><td>Multi-volume catalog of Greek coins from various museum collections worldwide.</td></tr>
            <tr><td><strong>DOC</strong></td><td>Dumbarton Oaks Collection</td><td>The primary reference for Byzantine coins.</td></tr>
            <tr><td><strong>RSC</strong></td><td>Roman Silver Coins</td><td>Seaby's reference focused on silver denominations.</td></tr>
          </tbody>
        </table>

        <p class="help-tip">💡 <strong>Getting started:</strong> You don't need to buy these expensive books right away. Websites like <a href="https://www.wildwinds.com" target="_blank" rel="noopener">WildWinds</a>, <a href="https://www.acsearch.info" target="_blank" rel="noopener">ACSearch</a>, and <a href="https://en.numista.com" target="_blank" rel="noopener">Numista</a> provide free searchable databases with images and catalog references.</p>

        <h4>How to Read a Catalog Reference</h4>
        <p><strong>Example: "RIC VII 162"</strong></p>
        <ul>
          <li><strong>RIC</strong> — The catalog (Roman Imperial Coinage)</li>
          <li><strong>VII</strong> — Volume number (Constantine I era)</li>
          <li><strong>162</strong> — The specific type number in that volume</li>
        </ul>
        <p>Always include the volume number when citing RIC, since type numbers restart in each volume.</p>
        <p>In coin details, use the structured <strong>Catalog References</strong> section for normalized references (catalog, volume, number, invoice number, authority URI). The legacy <code>referenceUrl</code>/<code>referenceText</code> fields are still supported for marketplace listing links and backward compatibility.</p>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Reading Inscriptions (Legends)</summary>
      <div class="help-content">
        <p>Roman coin inscriptions follow conventions that, once understood, make identification much easier. Legends are typically in Latin and read clockwise starting from the lower left.</p>

        <h4>Common Obverse Elements</h4>
        <table class="help-table">
          <thead>
            <tr><th>Abbreviation</th><th>Meaning</th><th>Example</th></tr>
          </thead>
          <tbody>
            <tr><td>IMP</td><td>Imperator (Commander/Emperor)</td><td>IMP CAESAR = Emperor Caesar</td></tr>
            <tr><td>CAES / CAESAR</td><td>Caesar (title)</td><td>—</td></tr>
            <tr><td>AVG</td><td>Augustus (title of honor)</td><td>IMP CAES TRAIANVS AVG</td></tr>
            <tr><td>P M / PONT MAX</td><td>Pontifex Maximus (chief priest)</td><td>—</td></tr>
            <tr><td>TR P / TRIB POT</td><td>Tribunicia Potestas (tribunician power) — renewed annually, useful for dating</td><td>TR P XV = 15th year of tribunician power</td></tr>
            <tr><td>COS</td><td>Consul (with number = which consulship)</td><td>COS III = third consulship</td></tr>
            <tr><td>P P</td><td>Pater Patriae (Father of the Fatherland)</td><td>—</td></tr>
            <tr><td>D N</td><td>Dominus Noster (Our Lord) — late Roman</td><td>D N CONSTANTIVS P F AVG</td></tr>
            <tr><td>P F</td><td>Pius Felix (Dutiful and Happy)</td><td>—</td></tr>
          </tbody>
        </table>

        <h4>Common Reverse Inscriptions</h4>
        <table class="help-table">
          <thead>
            <tr><th>Inscription</th><th>Meaning</th></tr>
          </thead>
          <tbody>
            <tr><td>S C (on bronzes)</td><td>Senatus Consulto — "by decree of the Senate"</td></tr>
            <tr><td>VICTORIA AVG</td><td>Victory of the Emperor</td></tr>
            <tr><td>PAX AVGVSTI</td><td>Peace of the Emperor</td></tr>
            <tr><td>CONCORDIA</td><td>Harmony/Agreement</td></tr>
            <tr><td>PROVIDENTIA AVG</td><td>Foresight of the Emperor</td></tr>
            <tr><td>GLORIA EXERCITVS</td><td>Glory of the Army — common on 4th century bronzes</td></tr>
            <tr><td>VOT V / X / XX</td><td>Votive offerings for 5/10/20 years of rule</td></tr>
            <tr><td>FEL TEMP REPARATIO</td><td>Restoration of Happy Times — common Constantine-era type</td></tr>
          </tbody>
        </table>

        <h4>Latin Reading Tips</h4>
        <ul>
          <li><strong>V = U</strong> — Latin had no letter U; AVGVSTVS = Augustus</li>
          <li><strong>I = J</strong> — No letter J; IVLIA = Julia</li>
          <li><strong>Ligatures</strong> — Letters sometimes share strokes (AE merged into Æ)</li>
          <li><strong>Retrograde</strong> — Very early coins sometimes have reversed letters</li>
          <li><strong>Worn legends</strong> — Use known text patterns and reference photos to fill gaps</li>
        </ul>

        <p class="help-tip">💡 <strong>Pro tip:</strong> The obverse legend usually tells you WHO (the emperor) and WHEN (via titles like TR P and COS numbers). The reverse tells you the MESSAGE (propaganda, virtues, military victories).</p>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Buying Your First Coins</summary>
      <div class="help-content">
        <h4>Where to Buy</h4>
        <ul>
          <li><strong>Established auction houses</strong> — CNG (Classical Numismatic Group), Heritage Auctions, Roma Numismatics, Nomos, Leu Numismatik. These guarantee authenticity.</li>
          <li><strong>Reputable dealers</strong> — Look for dealers who are members of professional organizations (ANA, IAPN, PNG).</li>
          <li><strong>VCoins / MA-Shops</strong> — Online marketplaces with vetted dealer storefronts. Return policies provide safety.</li>
          <li><strong>Coin shows</strong> — Great for handling coins and meeting dealers. Many shows have ancient coin sections.</li>
        </ul>

        <h4>What to Look for in a Coin</h4>
        <ul>
          <li><strong>Centering</strong> — A well-centered strike shows the full design. Off-center coins are less desirable (unless extremely rare).</li>
          <li><strong>Portrait quality</strong> — Sharp, detailed portraits are more appealing and valuable.</li>
          <li><strong>Full legends</strong> — Readable legends are important for identification and value.</li>
          <li><strong>Patina</strong> — Natural patina (surface coloring from age) is desirable. Don't clean ancient coins.</li>
          <li><strong>Eye appeal</strong> — Subjective but important. A coin should look attractive for its grade.</li>
          <li><strong>Provenance</strong> — Documented ownership history ("ex-collection") adds value and confirms legitimacy.</li>
        </ul>

        <h4>Budget Guidelines</h4>
        <table class="help-table">
          <thead>
            <tr><th>Budget</th><th>What You Can Expect</th></tr>
          </thead>
          <tbody>
            <tr><td>$10–$50</td><td>Late Roman bronzes (VG–F), Greek bronzes, uncleaned lots</td></tr>
            <tr><td>$50–$200</td><td>Better Roman bronzes (VF), common denarii (F–VF), interesting Greek bronzes</td></tr>
            <tr><td>$200–$500</td><td>Nice denarii (VF–EF), provincial silver, sestertii (F–VF)</td></tr>
            <tr><td>$500–$2,000</td><td>Choice denarii (EF+), Greek silver drachms, nice sestertii, gold fraction</td></tr>
            <tr><td>$2,000+</td><td>Gold aurei/solidi, Greek tetradrachms, rare types, top-grade coins</td></tr>
          </tbody>
        </table>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">⚠️ What to Watch Out For</summary>
      <div class="help-content">
        <h4>Forgeries & Fakes</h4>
        <p>Forgery has been a problem in numismatics for centuries. Modern fakes, especially from certain overseas workshops, can be very convincing. Here's how to protect yourself:</p>
        <ul>
          <li><strong>Buy from reputable sources</strong> — This is the #1 rule. Established auction houses and dealers guarantee authenticity. If a coin is fake, you get your money back.</li>
          <li><strong>Beware of "too good to be true"</strong> — A rare coin at a bargain price is almost certainly fake. If an eBay listing offers an EF Julius Caesar denarius for $50, walk away.</li>
          <li><strong>Check weight and diameter</strong> — Fakes often have incorrect weight or dimensions. Invest in a precision scale (0.01g) and calipers.</li>
          <li><strong>Study die characteristics</strong> — Genuine coins were struck from hand-engraved dies, creating subtle variations. Cast fakes often look "mushy" or have casting bubbles on the edge.</li>
          <li><strong>Look for casting seams</strong> — Genuine ancient coins were struck (hammered between two dies). Cast fakes may show a seam around the edge where the mold halves met.</li>
          <li><strong>Use resources</strong> — <a href="https://www.forumancientcoins.com/fakes/" target="_blank" rel="noopener">Forum Ancient Coins Fake Reports</a> and the Fake Ancient Coins group on Facebook are excellent resources.</li>
          <li><strong>Get it slabbed</strong> — For expensive purchases, consider NGC or PCGS authentication.</li>
        </ul>

        <h4>Cleaning</h4>
        <p class="help-warning">🚫 <strong>Never clean an ancient coin.</strong> Cleaning almost always reduces a coin's value, often dramatically. The natural patina that develops over centuries is considered part of the coin's character and beauty.</p>
        <ul>
          <li><strong>No brushing, polishing, or chemical dips</strong> — These remove patina permanently</li>
          <li><strong>Avoid "coin cleaning" products</strong> — They are designed for modern coins and will damage ancients</li>
          <li><strong>"Tooled" coins</strong> — Some fakes or cleaned coins have been re-engraved with a tool to restore detail. Look for unnatural scratch patterns in the fields.</li>
          <li><strong>Exception</strong> — Professional conservators can stabilize bronze disease or remove harmful deposits, but this is specialized work, not cleaning.</li>
        </ul>

        <h4>Common Scams & Pitfalls</h4>
        <ul>
          <li><strong>Uncleaned coin lots</strong> — Bulk lots of "uncleaned Roman coins" are a fun gamble for learning, but rarely contain anything valuable. Most will be low-grade Late Roman bronzes. Don't overpay.</li>
          <li><strong>eBay from certain regions</strong> — Be very cautious buying from regions known for producing fakes. Stick to sellers with excellent feedback and return policies.</li>
          <li><strong>"Guaranteed authentic"</strong> — Anyone can write this. A guarantee is only as good as the seller's reputation and return policy.</li>
          <li><strong>Over-grading</strong> — Sellers sometimes describe coins more generously than warranted. Learn to grade yourself by studying reference photos.</li>
          <li><strong>Misattributed coins</strong> — A coin labeled as one emperor may actually be another. Always verify the identification yourself using references.</li>
          <li><strong>Legal concerns</strong> — Be aware of cultural heritage laws. Buy coins with documented provenance or from dealers who comply with import/export regulations. Some countries restrict the export of antiquities.</li>
        </ul>

        <h4>Red Flags Summary</h4>
        <table class="help-table">
          <thead>
            <tr><th>🚩 Red Flag</th><th>What It Means</th></tr>
          </thead>
          <tbody>
            <tr><td>Price far below market value</td><td>Likely fake or misrepresented</td></tr>
            <tr><td>No return policy</td><td>Seller isn't confident in authenticity</td></tr>
            <tr><td>Casting bubbles on edge</td><td>Coin was cast (poured), not struck</td></tr>
            <tr><td>Wrong weight or dimensions</td><td>Common sign of modern reproduction</td></tr>
            <tr><td>Unnaturally shiny surface</td><td>Coin has been cleaned or is a modern fake</td></tr>
            <tr><td>Seam around the edge</td><td>Cast fake from a two-part mold</td></tr>
            <tr><td>"Museum quality" on eBay</td><td>Marketing language from unreliable sellers</td></tr>
          </tbody>
        </table>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Connecting AI Tools (External Tool Server)</summary>
      <div class="help-content">
        <p>The External Tool Server exposes your collection to external AI clients (OpenWebUI, LibreChat, n8n, and MCP-compatible clients) over a secure API. External clients can search your collection, retrieve statistics, and optionally propose updates through a two-phase confirmation workflow.</p>
        <p>For the complete technical walkthrough, see <code>docs/external-tool-server.md</code> in the repository.</p>

        <h4>For Admins</h4>
        <p><strong>Enabling the Server:</strong> The external tool server is disabled by default. To enable it:</p>
        <ol>
          <li>Navigate to <strong>Admin → System Settings</strong></li>
          <li>Toggle <strong>External Tool Server Enabled</strong> to ON</li>
          <li>The OpenAPI spec and tool endpoints become available immediately</li>
        </ol>
        <p>When disabled, all external tool requests fail with <code>503 Service Unavailable</code>. This default-off posture protects your instance until you explicitly choose to expose the API.</p>
        <p><strong>What to Tell Users:</strong> Once enabled, users can create scoped API keys in <strong>Settings → Data → API Keys</strong>. Each key can be read-only (safe for exploration) or read+write (for trusted automation). External writes are journaled and appear in each coin's activity log.</p>

        <h4>For Users</h4>
        <p><strong>Creating an API Key:</strong> To connect an external AI client to your collection:</p>
        <ol>
          <li>Navigate to <strong>Settings → Data → API Keys</strong></li>
          <li>Enter a descriptive name (e.g., "OpenWebUI Read-Only")</li>
          <li>
            Choose a capability:
            <ul>
              <li><strong>Read</strong> (default) — Allows queries: search, get coin, summaries, top coins</li>
              <li><strong>Read/Write</strong> — Allows queries plus update proposals and commits</li>
            </ul>
          </li>
          <li>Click <strong>Generate</strong></li>
          <li>Copy the displayed key (starts with <code>ak_</code>; shown only once)</li>
        </ol>
        <p><strong>Connecting to a Client:</strong> In your external AI client (OpenWebUI, LibreChat, n8n):</p>
        <ol>
          <li>
            Import the OpenAPI URL:
            <pre class="help-code">http://your-ancient-coins-host:8080/api/v1/tools/openapi.json</pre>
          </li>
          <li>
            Add a custom header:
            <ul>
              <li><strong>Header Name:</strong> <code>X-API-Key</code></li>
              <li><strong>Header Value:</strong> <code>ak_your_key_here</code></li>
            </ul>
          </li>
          <li>The six collection tools appear in your tool list</li>
        </ol>
        <p><strong>Write Safety:</strong> If you created a read+write key, external writes require two steps: first propose the change (returns a preview), then commit with <code>confirm=true</code>. This prevents accidental auto-writes during casual conversation. All commits are journaled.</p>

        <h4>For Developers</h4>
        <p><strong>Base Path:</strong> <code>/api/v1/tools/*</code></p>
        <p><strong>Authentication:</strong> Send your API key in the <code>X-API-Key</code> header.</p>
        <p><strong>Available Tools:</strong></p>
        <table class="help-table">
          <thead>
            <tr><th>Tool</th><th>Capability</th><th>Purpose</th></tr>
          </thead>
          <tbody>
            <tr><td><code>search_my_collection</code></td><td>read</td><td>Search coins by query string</td></tr>
            <tr><td><code>get_coin</code></td><td>read</td><td>Retrieve a single coin by ID</td></tr>
            <tr><td><code>collection_summary</code></td><td>read</td><td>Aggregate statistics (total coins, value)</td></tr>
            <tr><td><code>top_coins_by_value</code></td><td>read</td><td>Top coins sorted by current value</td></tr>
            <tr><td><code>propose_update</code></td><td>write</td><td>Phase 1: Propose allowlisted field changes</td></tr>
            <tr><td><code>commit_update</code></td><td>write</td><td>Phase 2: Commit with <code>confirm=true</code></td></tr>
          </tbody>
        </table>
        <p><strong>OpenAPI Spec:</strong> <code>GET /api/v1/tools/openapi.json</code> returns the full schema (unauthenticated, respects admin kill switch only).</p>
        <p><strong>MCP Compatibility:</strong> Wrap the OpenAPI spec with <a href="https://github.com/QuantGeekDev/mcpo" target="_blank" rel="noopener">mcpo</a> to expose tools to MCP clients (Claude Desktop, Cline):</p>
        <pre class="help-code">mcpo --openapi http://localhost:8080/api/v1/tools/openapi.json \
     --header "X-API-Key: ak_your_key_here"</pre>
        <p><strong>Security:</strong> All operations are user-scoped (tenant isolation). Rate limited at 50 requests/minute per key. External writes are restricted to the same allowlist as in-app updates (grade, currentValue, notes, tags, references). Identity fields are rejected.</p>
      </div>
    </details>

    <details class="help-accordion" :open="route.query.section === 'oidc'">
      <summary class="help-summary">Setting Up OIDC Login</summary>
      <div class="help-content">
        <p>OIDC login lets administrators add Microsoft Entra ID, Pocket ID, or another OpenID Connect provider while keeping local password recovery available.</p>
        <p>For the complete technical walkthrough, see <code>docs/oidc-setup.md</code> in the repository.</p>

        <h4>Before You Start</h4>
        <ul>
          <li>Keep at least one admin account with usable local credentials for recovery.</li>
          <li>Register both frontend callback URIs with the provider: login and account linking.</li>
          <li>Use scopes <code>openid profile email</code> unless your provider requires additional scopes.</li>
          <li>Store the client secret only in Admin Settings. It is write-only after saving.</li>
        </ul>

        <h4>Microsoft Entra ID</h4>
        <ol>
          <li>Create an app registration in the Entra admin center.</li>
          <li>Add Web redirect URIs for <code>/auth/oidc/callback/{providerId}</code> and <code>/settings/oidc/link/callback/{providerId}</code>.</li>
          <li>Enter the Tenant ID in Admin Settings; Aurearia derives the Microsoft issuer URL.</li>
          <li>Copy the Application client ID and the client secret <strong>Value</strong>, not the Secret ID.</li>
        </ol>

        <h4>Pocket ID</h4>
        <ol>
          <li>Create an OIDC client in Pocket ID.</li>
          <li>Set the issuer URL to your Pocket ID base URL.</li>
          <li>Register both frontend callback paths as allowed redirect URIs.</li>
          <li>Save the client ID and secret in Admin Settings, then run the provider test.</li>
        </ol>

        <p><strong>Account linking:</strong> Existing users should sign in locally, open Settings → Account, then link the provider. Matching email addresses are not merged automatically.</p>
      </div>
    </details>

    <details class="help-accordion">
      <summary class="help-summary">Helpful Resources</summary>
      <div class="help-content">
        <h4>Online Databases & References</h4>
        <ul>
          <li><a href="https://www.wildwinds.com" target="_blank" rel="noopener"><strong>WildWinds</strong></a> — Free database of ancient coin images organized by ruler and catalog reference</li>
          <li><a href="https://en.numista.com" target="_blank" rel="noopener"><strong>Numista</strong></a> — Collaborative coin catalog with detailed type information (integrated into this app)</li>
          <li><a href="https://www.acsearch.info" target="_blank" rel="noopener"><strong>ACSearch</strong></a> — Auction archives with realized prices — essential for valuation</li>
          <li><a href="https://www.coinarchives.com" target="_blank" rel="noopener"><strong>CoinArchives</strong></a> — Searchable archive of past auction lots with images and realized prices — great for valuation and attribution</li>
          <li><a href="https://rpc.ashmus.ox.ac.uk" target="_blank" rel="noopener"><strong>RPC Online</strong></a> — Free searchable database of Roman Provincial coins</li>
          <li><a href="https://www.forumancientcoins.com" target="_blank" rel="noopener"><strong>Forum Ancient Coins</strong></a> — Dealer, educational articles, and community discussion forum</li>
        </ul>

        <h4>Communities</h4>
        <ul>
          <li><a href="https://www.reddit.com/r/AncientCoins/" target="_blank" rel="noopener"><strong>r/AncientCoins</strong></a> — Active Reddit community for identification help, purchases, and discussion</li>
          <li><a href="https://www.cointalk.com/forums/ancient-coins.pair7/" target="_blank" rel="noopener"><strong>CoinTalk Ancient Coins</strong></a> — Long-running numismatic forum</li>
          <li><strong>Facebook groups</strong> — "Ancient Coins," "Fake Ancient Coins" (educational), and regional groups</li>
        </ul>

        <h4>Books for Beginners</h4>
        <ul>
          <li><em>Ancient Coin Collecting</em> by Wayne Sayles — The classic introduction to the hobby</li>
          <li><em>Roman Coins and Their Values</em> by David Sear — Identification and pricing guide (5 volumes)</li>
          <li><em>A Dictionary of Ancient Roman Coins</em> by John Melville Jones — Explains all the terminology</li>
          <li><em>Handbook of Greek Coinage</em> by Oliver Hoover — If Greek coins interest you</li>
        </ul>
      </div>
    </details>
  </section>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<style scoped>
/* Help Section */
.help-section {
  overflow: hidden;
  min-width: 0;
}

.help-section h2 {
  margin-bottom: 0.5rem;
}

.help-accordion {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  margin-bottom: 0.5rem;
  overflow: hidden;
}

.help-accordion[open] {
  border-color: var(--accent-gold-dim);
}

.help-summary {
  padding: 0.75rem 1rem;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  background: var(--bg-primary);
  color: var(--text-primary);
  list-style: none;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: background var(--transition-fast);
}

.help-summary::-webkit-details-marker {
  display: none;
}

.help-summary::before {
  content: '▸';
  font-size: 0.8rem;
  color: var(--text-muted);
  transition: transform 0.2s;
}

.help-accordion[open] > .help-summary::before {
  transform: rotate(90deg);
}

.help-summary:hover {
  background: var(--bg-card-hover, var(--bg-card));
}

.help-content {
  padding: 1rem;
  font-size: 0.9rem;
  line-height: 1.65;
  color: var(--text-secondary);
  overflow-x: auto;
}

.help-content h4 {
  color: var(--text-primary);
  margin: 1.25rem 0 0.5rem;
  font-size: 0.9rem;
}

.help-content h4:first-child {
  margin-top: 0;
}

.help-content p {
  margin-bottom: 0.75rem;
}

.help-content ul {
  margin: 0 0 0.75rem 1.25rem;
  padding: 0;
}

.help-content ol {
  margin: 0 0 0.75rem 1.25rem;
  padding: 0;
}

.help-content li {
  margin-bottom: 0.35rem;
}

.help-content a {
  color: var(--accent-gold);
  text-decoration: none;
}

.help-content a:hover {
  text-decoration: underline;
}

.help-tip {
  background: var(--accent-gold-glow, rgba(212, 175, 55, 0.08));
  border-left: 3px solid var(--accent-gold);
  padding: 0.6rem 0.85rem;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  font-size: 0.85rem;
  margin: 0.75rem 0;
}

.help-warning {
  background: rgba(231, 76, 60, 0.08);
  border-left: 3px solid #e74c3c;
  padding: 0.6rem 0.85rem;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  font-size: 0.85rem;
  margin: 0.75rem 0;
  color: var(--text-primary);
}

.help-table {
  width: 100%;
  border-collapse: collapse;
  margin: 0.75rem 0;
  font-size: 0.85rem;
  table-layout: fixed;
}

.help-table th,
.help-table td {
  padding: 0.5rem 0.65rem;
  text-align: left;
  border-bottom: 1px solid var(--border-subtle);
  word-wrap: break-word;
  overflow-wrap: break-word;
}

.help-table th {
  color: var(--text-muted);
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.help-table tr:last-child td {
  border-bottom: none;
}

.help-code {
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
  font-size: 0.8rem;
  color: var(--text-primary);
  overflow-x: auto;
  white-space: pre;
}
</style>
