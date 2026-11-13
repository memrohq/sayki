// Renders the Sayki app icon: swift icon.swift out.png
// "Say + Key": a keyboard keycap with a coral waveform on a dark tile.
import AppKit
let px = 1024.0
let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(px), pixelsHigh: Int(px), bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.saveGraphicsState(); NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)

// macOS icon grid: 824pt rounded tile centred on the 1024 canvas.
let tile = NSRect(x: 100, y: 100, width: 824, height: 824)
NSColor(red: 0.13, green: 0.13, blue: 0.15, alpha: 1).setFill()
NSBezierPath(roundedRect: tile, xRadius: 185, yRadius: 185).fill()

// Keycap: a darker base with a raised, slightly lighter top.
let cap = tile.insetBy(dx: 126, dy: 126)
NSGradient(starting: NSColor(white: 0.30, alpha: 1), ending: NSColor(white: 0.18, alpha: 1))!
    .draw(in: NSBezierPath(roundedRect: cap.offsetBy(dx: 0, dy: -27), xRadius: 121, yRadius: 121), angle: -90)
NSGradient(starting: NSColor(white: 0.26, alpha: 1), ending: NSColor(white: 0.21, alpha: 1))!
    .draw(in: NSBezierPath(roundedRect: cap.insetBy(dx: 27, dy: 27).offsetBy(dx: 0, dy: 11), xRadius: 99, yRadius: 99), angle: -90)

// Waveform in Sayki's coral-to-pink accent.
let accent = NSGradient(starting: NSColor(red: 1, green: 0.42, blue: 0.36, alpha: 1), ending: NSColor(red: 1, green: 0.30, blue: 0.55, alpha: 1))!
let heights: [Double] = [82, 176, 269, 176, 82]
let w = 44.0, gap = 33.0
let total = Double(heights.count) * w + Double(heights.count - 1) * gap
for (i, h) in heights.enumerated() {
    let x = cap.midX - total / 2 + Double(i) * (w + gap)
    accent.draw(in: NSBezierPath(roundedRect: NSRect(x: x, y: cap.midY - 5 - h / 2, width: w, height: h), xRadius: w / 2, yRadius: w / 2), angle: 0)
}

NSGraphicsContext.restoreGraphicsState()
try! rep.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: CommandLine.arguments[1]))
