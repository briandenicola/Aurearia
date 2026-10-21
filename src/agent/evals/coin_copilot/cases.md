# Coin Copilot eval cases

Source: `real` = your own question (screenshots/bug report), `repo` = from existing tests/fixtures, `synth` = written by me.
Collection, dealer, auction and price results come from fixtures (`collection.json`, `tests/fixtures/coin_copilot/specialists`);
specialist fixtures describe Domitian denarii, so any other query gets a no-match result. Web search and fetch run live.

| id | source | expected tools (one of each group) | web | outcome | judge also checks |
|---|---|---|---|---|---|
| collection-01 | repo | search_my_collection | no web | answer | Says plainly that no moose coins were found in the collection. |
| collection-02 | synth | collection_summary | no web | answer | Gives the coin count and the total current value from the collection summary. |
| collection-03 | synth | top_coins_by_value | no web | answer | Names the Athens tetradrachm, the Vespasian denarius and the Alexander III tetradrachm as the top three by recorded value. |
| collection-04 | synth | search_my_collection | no web | answer | Lists the Hadrian sestertius from the collection. |
| collection-05 | synth | collection_summary/search_my_collection/get_coin | no web | answer | general rubric only |
| curator-01 | repo | gap_analysis | no web | answer | general rubric only |
| curator-02 | synth | collection_summary + portfolio_review + gap_analysis | web optional | answer | Grounds the suggestions in what the collection actually holds. |
| curator-03 | synth | portfolio_review | no web | answer | general rubric only |
| curator-04 | repo | gap_analysis/search_my_collection | no web | answer | Notes that the collection holds no Flavian bronzes (the only Flavian coins are silver denarii). |
| shopping-01 | real | market_search | web only after specialist | answer | Reports that the configured dealer sources returned no matching listings rather than inventing any. |
| shopping-02 | real | market_search | web only after specialist | answer | Does not repeat the claim that shopping is outside its scope. |
| shopping-03 | synth | market_search | web only after specialist | answer | Presents the $275 CNG Domitian listing with its source URL. |
| shopping-04 | synth | market_search | web only after specialist | answer | general rubric only |
| auction-01 | repo | auction_search | web only after specialist | answer | general rubric only |
| auction-02 | synth | auction_search | web only after specialist | answer | Presents the upcoming Domitian lot with its estimate, sale date and source URL. |
| auction-03 | synth | auction_search | web only after specialist | answer | general rubric only |
| auction-04 | synth | similar_lots/auction_search | web only after specialist | answer | general rubric only |
| price-01 | synth | price_trends/similar_lots | web only after specialist | answer | Quotes completed-sale prices from the results and keeps their hammer price basis. |
| price-02 | synth | price_trends | web only after specialist | answer | general rubric only |
| price-03 | synth | search_my_collection/get_coin/top_coins_by_value + price_trends/similar_lots/auction_search | web only after specialist | answer | general rubric only |
| price-04 | real | price_trends/similar_lots/auction_search | web only after specialist | answer | States a specific maximum bid. Accounts for the 20% buyer's fee without mixing hammer and fee-inclusive prices. |
| deep-01 | synth | search_my_collection/get_coin + deep_analysis_handoff | no web | answer | Says the analysis was queued and points to the Deep Analysis review page. |
| deep-02 | synth | deep_analysis_handoff | no web | answer | general rubric only |
| deep-03 | synth | deep_analysis_handoff | no web | answer | Does not claim to have applied or accepted any proposed change. |
| deep-04 | synth | none required (never deep_analysis_handoff) | no web | clarify | Asks which of the several denarii in the collection is meant. |
| research-01 | real | none required | must fetch link | answer | Concludes that the obverse portrait is Caligula himself, not just a coin dated to his reign. Identifies the type as RPC I 1173 (or 1173a). Cites at least one source by name or URL. |
| research-02 | synth | none required | web optional | answer | Identifies Gaius and Lucius Caesars. Gives RIC I 207 (Lugdunum), or a closely related RIC I number with the range explained. |
| research-03 | synth | none required | no web | answer | Explains it as 'Judaea captured', commemorating the suppression of the Jewish revolt and the fall of Jerusalem in AD 70. |
| research-04 | synth | none required | web optional | answer | Points to the exergue mint mark (TR for Trier, LG or PLG for Lyon) as the deciding feature. |
| research-05 | synth | search_my_collection/get_coin | web optional | answer | Uses the details of the owned coin rather than answering about Hadrian sestertii in general. |

## collection-01
````
Do I have moose coins?
````

## collection-02
````
How many coins do I have and what's the whole collection worth?
````

## collection-03
````
What are my three most valuable coins?
````

## collection-04
````
Show me everything I have from Hadrian.
````

## collection-05
````
Which records in my collection are missing weight or diameter?
````

## curator-01
````
Compare my Roman coins and identify collection gaps.
````

## curator-02
````
What should I focus on buying next to round out my collection?
````

## curator-03
````
Give me a portfolio review. Am I too concentrated anywhere?
````

## curator-04
````
Compare my Flavian bronzes and identify gaps.
````

## shopping-01
````
Find Aurelian coins for sale online.
````

## shopping-02
Earlier user turn:
````
Find Aurelian coins for sale online.
````
Earlier assistant turn:
````
My role is limited to your personal coin collection. Would you like me to search?
````
````
I am not asking about my collection. I am asking your to find coins for sell online of any Aurelian coin
````

## shopping-03
````
I'd like a Domitian denarius with a Minerva reverse for under $300. Anything available?
````

## shopping-04
````
Find me a nice Athenian owl tetradrachm to buy, budget around $1,500.
````

## auction-01
````
Find Vespasian denarii at auction.
````

## auction-02
````
Any Domitian denarii coming up at auction soon?
````

## auction-03
````
What Roman Republican denarii are coming up in the next CNG e-auction?
````

## auction-04
````
Are there auction lots similar to my Domitian denarius?
````

## price-01
````
What do Domitian Minerva denarii sell for these days?
````

## price-02
````
Have Athenian owl tetradrachm prices gone up over the last few years?
````

## price-03
````
What's my Vespasian denarius worth now?
````

## price-04
Earlier user turn:
````
Take a look at this lot - [CORINTHIA, Corinth. Gaius (Caligula). AD 37-41. Æ As (19mm, 6.04 g, 12h). Marcus Bellius Proculus and Publius Vipsanius Agrippa, duoviri. Struck AD 37-38. Near VF. | Classical Numismatic Group](https://auctions.cngcoins.com/lots/view/4-N73O7H/corinthia-corinth-gaius-caligula-ad-37-41-as-19mm-604-g-12h-marcus-bellius-proculus-and-publius-vipsanius-agrippa-duoviri-struck-ad-37-38-near-vf). It says Caligula but I'm not sure if its really a coin of him or a coin struck during his reign. What do you think
````
Earlier assistant turn:
````
It is a portrait of Caligula himself, not just a coin dated to his reign. CNG catalogues it as RPC I 1173a: bare head of Caligula on the obverse, Pegasus on the reverse, issued at Corinth under the duoviri M. Bellius Proculus and P. Vipsanius Agrippa.
````
````
The bidding is at $225 now and there's a 20% buyer's fee on top. What maximum bid would you set?
````

## deep-01
````
Run a deep analysis on my Maximinus I denarius.
````

## deep-02
````
What's the status of deep analysis job 314?
````

## deep-03
````
Job 314 on my Maximinus I denarius (coin 42) got the mint wrong. Please rerun the deep analysis.
````

## deep-04
````
Run deep analysis on my denarius.
````

## research-01
````
Take a look at this lot - [CORINTHIA, Corinth. Gaius (Caligula). AD 37-41. Æ As (19mm, 6.04 g, 12h). Marcus Bellius Proculus and Publius Vipsanius Agrippa, duoviri. Struck AD 37-38. Near VF. | Classical Numismatic Group](https://auctions.cngcoins.com/lots/view/4-N73O7H/corinthia-corinth-gaius-caligula-ad-37-41-as-19mm-604-g-12h-marcus-bellius-proculus-and-publius-vipsanius-agrippa-duoviri-struck-ad-37-38-near-vf). It says Caligula but I'm not sure if its really a coin of him or a coin struck during his reign. What do you think
````

## research-02
````
Who is shown on the reverse of the Augustus denarius with the two standing princes, and what's its RIC number?
````

## research-03
````
What does IVDAEA CAPTA mean on Vespasian's coins?
````

## research-04
````
How do I tell a Constantinian follis struck at Trier from one struck at Lyon?
````

## research-05
````
Is my Hadrian sestertius a common type?
````
