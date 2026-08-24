# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Go 1.23 backend for **bread-machine.dev**, a production bread-recipe site (recipe scaler + sourdough
toggle) run by a single developer on a home server, tunnelled to the internet via Cloudflare and fronted
by Caddy. Data lives in Firestore; auth is Firebase Authentication.

The frontend is a **separate repo**, `../principles-of-baking` (React + Vite). Do not edit files there
from this repo — it's out of scope for backend work. See "API contract" below for what it expects from
this server.

Module path: `github.com/BreadBrand/breadmachine`.

## Commands

- Build: `go build ./...`
- Test all: `go test ./...`
- Test one package: `go test ./internal/parser/...`
- Test one case: `go test ./internal/parser -run TestParseIngredients_Xyz`
- Vet: `go vet ./...`
- Live smoke test (hits real Firestore/Firebase Auth, no writes, no emulator): `go test -tags=live ./handlers/... -run TestLive_FirebaseAdminV4_Smoke` — requires the real service account key; not run as part of `go test ./...`.
- Emulator-backed handler tests (`handlers/recipe_emulator_test.go`): `./scripts/test-emulators.sh` — spins up local Firestore + Auth emulators via `npx firebase-tools` (no global install, no real GCP project/credentials) and runs the handler tests against them. Requires Node/npm and a JDK on PATH. Skipped automatically (not failed) by plain `go test ./...` when the emulators aren't running.
- Run locally: `go run main.go` — **requires** a Firebase service account key at the hardcoded path
  `/etc/breadmachine/serviceAccountKey.json` (see `handlers/firebase.go`); `InitFirebase` fatals without
  it, so the server won't start.
- Deploy: `./deploy-api.sh` — cross-compiles for linux/amd64, rsyncs the binary to
  `bbashein@gizmo:/opt/breadmachine-api`, and restarts `breadmachine.service` over SSH. This ships
  straight to the production server; treat it as a real deploy, not a build step.

## Architecture

`main.go` registers five routes on the stdlib Go 1.22+ pattern-matching mux, one pattern per
method+path combination (method-specific patterns take precedence over same-path wildcards, and a
mismatched method on an otherwise-matching path gets an automatic `405` + `Allow` header from the mux
itself — no manual method switches needed in handlers):

- `GET /api/recipes` — list, no auth → `handlers.GetAllRecipes`
- `POST /api/recipes` — create, Firebase-auth required → `handlers.CreateRecipe`
- `POST /api/recipes/parse` — Firebase-auth required → `handlers.ParseHandler`
- `GET /api/recipes/{id}` — single by id, no auth → `handlers.GetRecipe`
- `DELETE /api/recipes/{id}` — Firebase-auth required, caller must own the recipe → `handlers.DeleteRecipe`

**`handlers/`** — HTTP layer, one file per concern:
- `firebase.go` — Firestore/Auth client init, direct Firestore reads/writes against the `Recipes`
  collection, and `authenticate(r)` — the shared Bearer-token verification helper used by every
  protected route. It returns `(uid, ok)`; each caller still renders its own failure response (plain
  `http.Error` in `CreateRecipe` vs. the JSON envelope in `ParseHandler`/`DeleteRecipe`) since the two
  formats coexist in this codebase — see the error-envelope note in the API contract section.
- `recipe.go` — route handlers plus `normalizeIngredients`, which runs on every create: assigns a UUID
  per ingredient, canonicalizes the unit string, and fills in `Grams` (via `unitToGrams` for
  fixed-density units, `utility.LookupCountWeight` for "count" units, or leaves it if already set).
- `normalize.go` — `convertToGrams`: once a recipe has *any* mass-unit ingredient ("gram-dominant"), every
  volume-unit ingredient with a known density gets rewritten to grams so the recipe has one consistent
  unit system.
- `parse.go` — thin wrapper around `internal/parser.Parse`; translates parser sentinel errors into the
  `{error, message}` JSON shape (see API contract).

**`models/recipe.go`** — Firestore document shapes (`Recipe`, `Ingredient`, `Meta`) plus
`CalculateBakerPercentages`. Baker's percentage = ingredient grams ÷ sum of grams of "base" ingredients
(name contains flour/lentil/oat/cauliflower/chickpea/tapioca — covers grain-free and legume-based
recipes), restricted to dough-phase ingredients (dough/scald/tangzhong/yudane/starter build/levain/final
dough). Recipes with no matching base ingredient get all-zero percentages. Call with `DoughIngredients`
only — `OtherIngredients` (toppings/fillings) are never part of baker's math.

**`internal/parser/`** — pure text → `RecipeDTO` pipeline, no Firestore/HTTP/auth/gram-math dependency by
design (see boundary rule below). `Parse()` in `parser.go` runs six stages in order, one file each:

1. `normalise.go` — strip HTML/markdown, unescape entities, normalize curly quotes and unicode fractions,
   combine mixed numbers ("2 3/4" → "2.75"), drop browser/print/nutrition-block/attribution noise, expand
   inline `Key: Value Key: Value` metadata lines.
2. `sections.go` — splits into title / description / metadata lines / ingredient groups (each tagged with
   a phase like `"starter build"`, `"tangzhong"`, or a free-form subsection name) / instruction lines.
3. `ingredients.go` — parses each ingredient line into name/quantity(string)/unit; routes lines to dough
   vs. other by group phase, with any line containing "topping" forced to `other` regardless of phase.
4. `instructions.go` — numbered/bulleted step extraction, disclaimer-line filtering (e.g. "results may
   vary").
5. `metadata.go` — servings/prepTime/cookTime/additionalTime extraction; time values normalized to
   integer minutes.
6. `confidence.go` — per-field 0.0–1.0 confidence score, using pessimistic min-stacking when multiple
   conditions hit the same field. Frontend should flag scores below 0.6 for user review.

`units.go` (KnownUnits / CanonicalUnit) and `yeast.go` (supported vs. unsupported yeast/leavener phrases)
are shared lookup tables used across the ingredient and confidence stages.

**`utility/ingredient_densities.go`** — ordered (specific-before-generic keyword match) density table in
g/mL, plus a count-unit weight table. Consumed only by the save path (`handlers/recipe.go`
`normalizeIngredients`) — never by `internal/parser`.

**`cmd/`** — one-off operational scripts (`backfill-density`, `backfill-grams`, `backfill-units`,
`fix-lentil-bread`, `migrate`) that connect directly to the production Firestore `Recipes` collection
using the same hardcoded service-account path as the server. These are run manually against prod, not
part of the running application — treat any change here as a data-migration change, and confirm intent
before running one.

### Architectural boundary: `internal/parser`

Per the parser engine design docs (`.claude/recipe_parser_engine_*.md`, gitignored — local reference
only): `internal/parser` must never import Firestore, HTTP context, auth tokens, `userId`, the density
table, or baker's-percentage logic. It only knows text/regexp/stdlib and its own DTOs
(`RecipeDTO`/`SectionMap`/`IngredientDTO`). All gram calculation, density lookups, and baker's-percentage
math happen afterward, in the save path. If a change to the parser needs any of those, the logic belongs
in `handlers/recipe.go` or `models/recipe.go` instead.

### Fixed: single-recipe GET/DELETE routing and delete auth

`RecipeHandler` used to extract the path id via `strings.TrimPrefix(r.URL.Path, "/recipes/")` while the
route was actually mounted at `/api/recipes/`, so `id` never stripped correctly and neither GET-by-id nor
DELETE could resolve a document. Fixed by switching to Go 1.22 path parameters
(`GET /api/recipes/{id}`, `DELETE /api/recipes/{id}` → `r.PathValue("id")`), split into
`handlers.GetRecipe` and `handlers.DeleteRecipe`. `DeleteRecipe` also now requires a valid Firebase token
and checks that `recipe.UserID` matches the token's UID before deleting (`403 FORBIDDEN` otherwise) —
previously it had no auth check at all. `GetRecipe` stays unauthenticated, matching the list endpoint.
No frontend change was needed for this — `principles-of-baking` doesn't call either endpoint yet, so this
was dead/broken code becoming live rather than a contract change. If you build a delete UI, it'll need to
send `Authorization: Bearer <token>` like `UseCreateRecipe.ts` already does.

### Known gap: no recipe update endpoint

There is no PUT/PATCH route anywhere in `main.go` — only create (`POST /api/recipes`), list
(`GET /api/recipes`), single-recipe GET/DELETE, and parse. A saved recipe can currently only be created
or deleted, never edited. If an "edit a saved recipe" feature is planned, that route and handler don't
exist yet and need to be built from scratch.

## API contract (for the frontend repo)

The frontend types in `principles-of-baking/src/types/{dto,models}.ts` don't fully match what this server
sends. Concrete mismatches, verified against the frontend hooks (`UseFetchRecipes`, `UseCreateRecipe`,
`useParseRecipe`):

- **`POST /api/recipes/parse` → `RecipeDTO.doughIngredients[].quantity` is a string**, the raw token as it
  appeared in the source text (e.g. `"1/2"`, `"2.75"`), not a number — the frontend must parse it before
  doing arithmetic. `types/dto.ts`'s `IngredientDTO.quantity: number` is wrong for this response.
- **`RecipeDTO.prepTime` / `cookTime` / `additionalTime` are integers in minutes**, not strings.
  `types/dto.ts` types them as `string`; `types/models.ts`'s `Meta` has the same mismatch for the
  persisted `Recipe.meta` fields (also integers server-side, per `models/recipe.go`).
- `RecipeDTO.otherIngredients` can be JSON `null` (Go nil slice) when empty — already modeled correctly
  in `types/dto.ts`.
- `Ingredient.grams` is always present, lowercase, computed server-side at save time in every case — never
  trust or send a client-computed value for it.
- `Ingredient.bakerPercentage` is computed server-side and is `0` when the recipe has no ingredient
  matching a base keyword (flour/lentil/oat/cauliflower/chickpea/tapioca).
- Error responses from `/api/recipes/parse` and `DELETE /api/recipes/{id}` are
  `{"error": "CODE", "message": "human string"}`. Parse codes: `UNAUTHORIZED`, `INVALID_JSON`,
  `INPUT_EMPTY`, `INPUT_TOO_LARGE`, `PARSE_FAILED` (422 — no ingredients or instructions found),
  `INTERNAL_ERROR`. Delete codes: `UNAUTHORIZED`, `NOT_FOUND`, `FORBIDDEN` (caller doesn't own the
  recipe), `INTERNAL_ERROR`. `POST /api/recipes` (create) and `GET /api/recipes/{id}` instead return
  plain-text bodies via `http.Error` — don't try to `JSON.parse()` those.
- Auth: `Authorization: Bearer <Firebase ID token>` is required and verified server-side for
  `POST /api/recipes`, `POST /api/recipes/parse`, and `DELETE /api/recipes/{id}` (which additionally
  requires the caller to own the recipe, or `403 FORBIDDEN`). `GET /api/recipes` (list) and
  `GET /api/recipes/{id}` (single) are unauthenticated by design.

**Unit contract gotcha:** the frontend's `UNITS` constant (`types/dto.ts`) offers `"Tbls"` as a selectable
unit, but neither the backend's canonical-unit table (`internal/parser/units.go` `unitCanonicals`) nor its
gram-conversion table (`handlers/recipe.go` `unitToGrams`) recognize `"tbls"` — only `"tbsp"` (and the
legacy `"tbs"`). A recipe created with unit `"Tbls"` silently falls back to a ×1 gram multiplier (1 Tbls
stored as 1 gram instead of 15), which corrupts `yieldGrams` and every baker's percentage on that recipe.
Either change the frontend option to `"tbsp"`, or add a `"tbls"` alias in `unitCanonicals`.
