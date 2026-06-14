# Validating the LMU adapter against the live game

The LMU adapter is verified by automated tests and compiles for Windows, but it
has never read the actual game. This guide is the on-PC validation pass: get real
data flowing, then confirm the values are correct. Everything here runs on the
**Windows gaming PC**.

## 1. Install the shared-memory plugin (prerequisite)

LMU reads/writes the community **rFactor2 Shared Memory Map Plugin** — the same
plugin rFactor 2 uses. Without it, PitMate sees nothing.

1. Download the latest **`rFactor2SharedMemoryMapPlugin64.dll`** from the
   plugin's GitHub releases:
   <https://github.com/TheIronWolfModding/rF2SharedMemoryMapPlugin/releases>
2. Copy the DLL into LMU's `Plugins` folder, typically:
   ```
   ...\Steam\steamapps\common\Le Mans Ultimate\Plugins\rFactor2SharedMemoryMapPlugin64.dll
   ```
3. Launch LMU once with the DLL present, then quit. This makes the game add the
   plugin to its config file:
   ```
   ...\Steam\steamapps\common\Le Mans Ultimate\UserData\player\CustomPluginVariables.JSON
   ```
4. Open that JSON file and make sure the plugin block is **enabled** (note the
   leading space in `" Enabled"` — that quirk is real and required):
   ```json
   "rFactor2SharedMemoryMapPlugin64.dll":{
     " Enabled":1
   }
   ```
5. Launch LMU and load any session (a test/practice session is fine).

> **Compatibility note.** The LMU wiki warns that plugin compatibility depends on
> LMU's build version. If PitMate connects but values look garbled or frozen,
> suspect a plugin/LMU version mismatch first — grab the newest plugin release.

## 2. Run PitMate on the gaming PC

**Recommended (simplest): install Go and run from source.** No file copying, and
it's the same workflow as the rest of the project.

1. Install Go (<https://go.dev/dl/>), then:
   ```bash
   git clone https://github.com/stevirn/PitMate.git
   cd PitMate/backend
   go run . -dump
   ```
   `-dump` prints a once-a-second summary to the console — the easiest way to
   validate solo while driving. Add `-static ../frontend/dist` (after
   `npm run build` in `frontend/`) if you also want the browser cockpit.

**Alternative: a prebuilt .exe.** If you'd rather not install Go on the gaming
PC, I can cross-compile a Windows `.exe` for you to copy over (USB / network
share). Just ask and I'll produce it.

You do **not** need `-mock` here — that's only for development without the game.

## 3. What you should see

- Start PitMate, then load an LMU session. The console should switch from
  `no live data …` to dump blocks like:
  ```
  ── PitMate dump ──
    game=Le Mans Ultimate session=practice track="..." flag=green sc=none ...
    pos P1 (class P1) lap=... last=... best=... gapAhead=... gapBehind=...
    speed=... gear=... rpm=... fuel=...L (... laps) bias=...%F oil=... water=...
    tires°C FL/FR/RL/RR=.../.../.../...  wear%=.../.../.../...  compound="..."
  ```
- In a browser (if serving the UI): the connection dot goes **green** and the
  Live Data tab fills in.

## 4. Validation checklist

Drive a few laps and compare the dump/UI to the in-game HUD. ✅ = trust it,
⚠️ = inferred mapping, most likely to be wrong; if one is off, tell me the
observed-vs-expected and I'll fix it in `backend/adapters/lmu/mapping.go`.

| Value | How to check | Risk |
|---|---|---|
| Speed / gear / RPM | Match the in-car HUD | ✅ low |
| Fuel (litres) | Match the fuel readout | ✅ low |
| Oil / water temp | Match engine temps | ✅ low |
| Tire temps (°C) | Plausible (~70–110°C hot); not ~350 (=Kelvin not converted) | ✅ low |
| Tire wear % | **100% = new, dropping with laps.** If it shows ~0% on fresh tires, the wear direction is inverted | ⚠️ check |
| Brake bias %F | Match the in-car brake-bias setting | ✅ low |
| Position / gaps | Match the standings; gapAhead/Behind sane | ✅ low |
| Class position | Correct rank within your class | ⚠️ check |
| Lap / sector times | Match timing; sectors sum to the lap | ✅ low |
| **Session type** | practice/qualifying/race matches reality | ⚠️ check |
| **Flag / safety car** | Green when racing; yellow/SC/red when they happen | ⚠️ check |
| **Pit status** | none → approach → stopped → exit across a stop | ⚠️ check |
| Track wetness / rain | 0 in the dry; rises in the wet | ⚠️ check |
| Compound | Matches fitted tire | ⚠️ check |

Expected gaps (already known, not bugs): car number is blank, TC/ABS/ARBs/engine
map and detailed damage are empty, and fuel-per-lap / laps-remaining stay 0 until
the stateful enrichments are built (see `PROGRESS.md`).

## 5. Reporting back

For anything that looks wrong, the most useful thing is: the field, what PitMate
showed, and what the game showed (a screenshot of the dump console next to the
HUD is ideal). With that I can correct the specific mapping quickly.
