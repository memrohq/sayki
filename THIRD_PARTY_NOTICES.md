# Third-party notices

Sayki includes or downloads the following third-party components. Each remains
under its own license; the full license texts are available at the linked sources.

## Bundled with the app

| Component | License | Source |
|---|---|---|
| sherpa-onnx (Go bindings and C API library) | Apache-2.0 | https://github.com/k2-fsa/sherpa-onnx |
| ONNX Runtime | MIT | https://github.com/microsoft/onnxruntime |
| Wails v3 | MIT | https://github.com/wailsapp/wails |
| malgo / miniaudio | Unlicense / public domain (miniaudio also MIT-0) | https://github.com/gen2brain/malgo |
| modernc.org/sqlite | BSD-3-Clause | https://gitlab.com/cznic/sqlite |
| React, React DOM | MIT | https://github.com/facebook/react |
| @wailsio/runtime | MIT | https://github.com/wailsapp/wails |
| Tailwind CSS (build-time) | MIT | https://github.com/tailwindlabs/tailwindcss |

## Speech models (downloaded on request, not bundled)

| Model | License | Attribution |
|---|---|---|
| Parakeet TDT 0.6B v2 | CC-BY-4.0 | Model weights © NVIDIA Corporation, https://huggingface.co/nvidia/parakeet-tdt-0.6b-v2. Converted to ONNX by the sherpa-onnx project. Sayki runs the converted model unmodified. |
| Streaming Zipformer models (English, Chinese + English, French, Korean) | Apache-2.0 | k2-fsa / icefall and sherpa-onnx contributors, https://github.com/k2-fsa/sherpa-onnx |
| English punctuation model | Apache-2.0 | sherpa-onnx contributors, https://github.com/k2-fsa/sherpa-onnx |

Apple speech recognition is provided by macOS's Speech framework and is not
redistributed.
