# Match'it — Go + Ebiten Port

This repository is a faithful Go/Ebiten port of the Atari ST puzzle game “Match'it”. All original mechanics are preserved: board generation and progression, matching rules (identical tiles, seasons-with-seasons, flowers-with-flowers), the 2-bend path constraint, help logic, timing/bonuses, and removal animations. YM music plays via the bundled `Chambers of Shaolin - Trapped in China.ym`.

## How to Play
- Clear the board by selecting pairs of tiles that can be connected with a path containing at most two right-angle turns.
- Identical tiles pair; seasons pair with seasons; flowers pair with flowers.
- Controls: Left click to select; `H` for help; `R` restart level; `P` pause; `ESC` return to menu; `M`/`m` mute/unmute music.
- Score carries across levels; unused help grants a bonus help and contributes to the end-of-level bonus.

## Build & Run
- Launch the game:
   ```sh
   go run ./cmd/matchit
   ```

## Project Layout
- `cmd/matchit/` — game entrypoint.
- `cmd/convert-assets/` — converts original assets to PNG under `assets/png/` (levels, tiles, UI plates, fonts, help plates, remove masks).
- `internal/game/` — rendering, input, state machine (splash, menu, play, summaries, highscores, instructions), audio control.
- `internal/logic/` — level data, RNG/permutation, matching and pathfinding.
- `internal/assets/` — decoders used by the converter and runtime loaders for PNG assets.
- `assets/png/` — runtime-ready PNGs (generated); includes `malakhsoftware-pixel.png` for the splash.

## Notes & Development
- The game only reads PNG assets at runtime; keep originals in `old/` and out of git.
- Fonts: menu uses the original FONT2 PNGs; instructions use a readable system font.
- Music stays running across screens; mute is global.
- Manual checks after changes: asset converter runs cleanly; splash -> menu transition occurs at 3s; menu labels align on red bars; matching rules and help suggestions obey the 2-bend path; pause/mute work; score persists; highscores entry handles typing correctly.

## Credits
- Original game by Delta Force : code by New Mode, graph by Slime (Dec 1990).
- Conversion by Olivier Houte (Malakh Software). This port keeps data and behavior true to the Atari ST release while modernizing rendering and audio through Ebiten and Go.
