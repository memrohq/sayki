# Sayki

Private voice typing for macOS. Press a shortcut, speak, and your words are typed
wherever your cursor is: any app, any text field. Speech recognition runs entirely
on your Mac. No account, no cloud, no word limits.

## Features

- **Works everywhere you can type**: editors, browsers, chat apps, terminals.
- **On-device only**: audio never leaves your Mac.
- **Choice of speech engines**
  - **Apple speech**: macOS's built-in on-device recognizer. Nothing to download.
  - **NVIDIA Parakeet TDT 0.6B**: the most accurate English option; transcribes the whole recording when you stop, with punctuation and capitalization.
  - **Zipformer** streaming models: English, Chinese + English, French, Korean.
- **Toggle or push-to-talk** with a global shortcut (default <kbd>⌃</kbd><kbd>⌥</kbd><kbd>Space</kbd>).
- **Floating overlay** with a live waveform and transcript preview; drag it anywhere, on any display.
- **Menu bar app**: no Dock icon.

## Requirements

- macOS 13 or later (Apple Silicon or Intel)
- To build: Go 1.26+, Node.js 20+, Xcode Command Line Tools

## Build and run

```sh
make app   # builds the frontend, compiles, bundles and signs build/bin/Sayki.app
make run   # builds, then opens the app
make test  # go test -race ./...
```

`make app` signs with the first code-signing identity in your keychain, or ad-hoc
(`-`) if there is none. A stable identity keeps macOS privacy permissions across
rebuilds; with ad-hoc signing you will be asked again after each build.

On first launch, grant **Microphone** and **Accessibility** access (Accessibility is
what lets Sayki type into other apps), then pick a speech model in
**Settings → Models**.

## Privacy

Sayki only records while you hold or toggle the shortcut. Transcripts are never
logged. History is off by default and stored only in a local SQLite file. Models
are downloaded from the [sherpa-onnx releases](https://github.com/k2-fsa/sherpa-onnx/releases)
only when you click Download.

## License

Sayki is dual-licensed:

- **Open source:** [GNU AGPL-3.0](LICENSE). You may use, modify and redistribute
  Sayki, including commercially, as long as you comply with the AGPL. In
  particular, if you distribute Sayki or a modified version, or make it available
  to users over a network, you must release your complete source code under the AGPL.
- **Commercial license:** if you want to ship Sayki, or a product built on it,
  without the AGPL's source-sharing obligations, buy a commercial license.
  Contact **[subal@memro.co](mailto:subal@memro.co)**.

Third-party components keep their own licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## Contributing

Contributions are welcome. Because Sayki is dual-licensed, contributors must sign a
Contributor License Agreement before a pull request can be merged. See
[CONTRIBUTING.md](CONTRIBUTING.md).
