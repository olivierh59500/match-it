# Match'it — Go + Ebiten Port

This repository is a faithful Go/Ebiten port of the Atari ST puzzle game “Match'it”. All original mechanics are preserved: board generation and progression, matching rules (identical tiles, seasons-with-seasons, flowers-with-flowers), the 2-bend path constraint, help logic, timing/bonuses, and removal animations. YM music plays via the bundled `Chambers of Shaolin - Trapped in China.ym`.

## How to Play
- Clear the board by selecting pairs of tiles that can be connected with a path containing at most two right-angle turns.
- Identical tiles pair; seasons pair with seasons; flowers pair with flowers.
- Controls: Left click to select; `H` for help; `R` restart level; `P` pause; `ESC` return to menu; `M`/`m` mute/unmute music.
- Touch controls: tap tiles to select them, tap the help counter for a hint, and use the `MENU`, `PAUSE`, `RESTART`, and `MUSIC` bar below the board. Name entry has a complete on-screen keyboard.
- Score carries across levels; unused help grants a bonus help and contributes to the end-of-level bonus.

## Build & Run
- Launch the game:
   ```sh
   go run ./cmd/matchit
   ```

## Android

The Android launcher is configured for phones and tablets in sensor-landscape mode. The fixed 640x400 logical canvas scales uniformly to the available display, and all runtime assets are embedded in the Go library.

Prerequisites: Go, `ebitenmobile`, Android SDK/API 36, Android NDK, JDK 17, and USB debugging enabled on the device. `ANDROID_HOME` or `ANDROID_SDK_ROOT` can be used for a non-standard SDK location.

- Build, install, and launch the debug application on the connected device:
   ```sh
   ./scripts/run-android.sh
   ```
- Build only the Go Android library:
   ```sh
   ./scripts/build-android-aar.sh
   ```
- Build only the APK:
   ```sh
   cd android
   ./gradlew :app:assembleDebug
   ```

The generated APK is `android/app/build/outputs/apk/debug/app-debug.apk`. High scores are saved in the application's private Android files directory.

## Project Layout
- `cmd/matchit/` — game entrypoint.
- `cmd/convert-assets/` — converts original assets to PNG under `assets/png/` (levels, tiles, UI plates, fonts, help plates, remove masks).
- `mobile/` — Ebitengine mobile binding package.
- `android/` — native Android launcher and Gradle project.
- `scripts/` — Android binding/build/install helpers.
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
