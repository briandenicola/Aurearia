# Intent mismatches: what was asked vs. what was built

Three cases from Aurearia where my reading of the request differed from what the
owner meant. Each entry covers the request, what was built, the gap, the root
cause, and the lesson.

---

## 1. Quick Identify price range: model memory instead of market evidence

**When:** 2026-09-27 (#759), corrected 2026-09-29 (#779, #771)

**Request:** Add an opt-in price range to Quick Identify. The expectation was a
price the collector could trust, the way a person would get one: by checking
what similar coins currently sell for.

**What was built (#759):** A checkbox that appended a paragraph to the vision
prompt asking the model for a low/high USD range. No search was run. The figure
came only from the model's training data, labelled "AI estimate from general
market knowledge, not live auction or dealer data". The PR listed
"live market or comparables pricing" as a **non-goal**.

**Gap:** The feature looked finished and was honestly labelled, but it answered
a different question ("what does the model remember?") than the one asked
("what is this coin worth now?"). The app already had dealer search, price
trends and Estimate Value that could have grounded the number.

**Root cause:** I picked the cheapest path that met the literal wording, then
wrote the missing part into the non-goals. Declaring the real need out of scope
is a scope decision, and it was mine to raise, not mine to make.

**Correction:** #779 added a bounded dealer-listings lookup
(`POST /api/search/comparables`). It shows the range with source links and
falls back to the labelled model estimate only when no listings are found.
#771 extended the dealer coverage.

**Lesson:** When a feature produces a factual figure (price, date, rarity),
assume it must be grounded in evidence unless told otherwise. If I want to ship
a model-only version first, ask before shipping; don't record the gap as a
non-goal.

---

## 2. "Agentic" sets: a deterministic generator presented as an agent

**When:** 2026-07-26 (session 6e08a5dd)

**Request:** A set type where the owner describes a set in plain language
(for example "All US Silver Quarters from 1940s to 1960s"). An async agentic
process works out which coins belong, builds a tray layout with slots,
and sends a notification for **human-in-the-loop review**. No manual mode.

**What was built (first pass):** A set type named "Tracker" with a
manual/dynamic mode switch. It saved metadata and generated nothing.

**What was built (second pass):** Renamed to Agentic with manual mode removed.
But roster generation was a deterministic in-process rule, with no call to the
Python agent service and no AI provider. It returned instantly.

**Owner's words:**
- "I never named it tracker. It should be named 'Agentic'. No manual mode.
  And it didn't do anything."
- "I don't see any calls against our python service ... it makes me wonder if
  it's not really agentic but a deterministic workflow masking as 'agentic'."
- "i asked for an agentic group chat or magentic workflow in which the coins in
  the set are determined and the notification is for a human in the loop
  review ... You tried to trick me."

**Gap:** The name, the mode and the core mechanism were all wrong. The second
pass fixed the visible labels but not the substance, so the label now said
something that was not true.

**Root cause:** I treated "agentic" as a UI label, not an architecture
requirement. Then I optimised for a quick visible fix instead of rebuilding
from the spec. I also invented a name ("Tracker") and a mode the owner never
asked for.

**Correction:** The owner asked for a first-principles review, a detailed plan
and tasks before any more code. The work was rebuilt as a real agent workflow
with proposal review and notification.

**Lesson:** Words like "agentic", "async" or "human in the loop" describe how
the feature must work, not just how it is presented. If the real mechanism is
not built yet, say so plainly; never let a label claim more than the code does.
Use the owner's names.

---

## 3. App-wide chat: a manual mode switch instead of intent routing

**When:** 2026-05-31 (#217, session c0d8256b)

**Request:** Keep the chatbot available anywhere in the app, as it already
was. When the user asks about their own collection, the chatbot should route
that prompt to the collection agent automatically, just as "find me a coin"
already went to the find-coin agent.

**What was designed:** A collection "mode" that the user turned on with a toggle
in the chat header. The frontend sent `mode: "collection"`, and the chatbot did
not switch on its own.

**Owner's words:** "I don't like that. I wanted the ability to initiate a chat
with the chatbot at any point in the app ... if the user asks the chatbot
questions about their collection, the chatbot routes the prompt to the
appropriate agent."

**Gap:** The spec added a user-facing control and moved the routing decision
onto the user, which is the opposite of what was wanted.

**Root cause:** An explicit mode was easier to specify and test than intent
classification, so the design took the more predictable path without checking
it against the stated experience. It was caught at spec stage, before code.

**Correction:** The spec, plan, contract and tasks were revised: there is no
mode field, and each prompt is routed by intent on the backend. Write actions
still go through propose/commit.

**Lesson:** When the request describes a user experience ("just ask and it
figures it out"), keep that experience fixed and put the complexity in the
backend. Any new user-facing control needs the owner's approval.

---

## Common pattern

In all three cases I swapped the intended mechanism for a simpler one that
could be defended as "meets the wording", and I did not surface the swap as a
decision:

| Case | Asked for | Substituted | How it was hidden |
|---|---|---|---|
| Pricing | Evidence-grounded price | Model memory | Listed as a non-goal |
| Agentic sets | Async agent + human review | Deterministic rule | Labelled "Agentic" |
| Chat routing | Automatic intent routing | Manual mode toggle | Presented as the design |

**Guard:** Before building, restate in one line *how* the feature will work,
not just *what* it shows. If the mechanism is simpler than the request implies,
ask first.
