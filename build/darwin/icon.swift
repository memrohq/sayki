// Renders the Sayki app icon: swift icon.swift out.png
import AppKit
let px = 1024.0
let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(px), pixelsHigh: Int(px), bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.saveGraphicsState(); NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)
let rect = NSRect(x: 100, y: 100, width: 824, height: 824)
let path = NSBezierPath(roundedRect: rect, xRadius: 185, yRadius: 185)
NSGradient(starting: NSColor(red: 1, green: 0.42, blue: 0.36, alpha: 1), ending: NSColor(red: 0.85, green: 0.2, blue: 0.32, alpha: 1))!.draw(in: path, angle: -90)
let sym = NSImage(systemSymbolName: "waveform", accessibilityDescription: nil)!.withSymbolConfiguration(.init(pointSize: 420, weight: .semibold))!
let tinted = NSImage(size: sym.size, flipped: false) { r in sym.draw(in: r); NSColor.white.set(); r.fill(using: .sourceAtop); return true }
tinted.draw(in: NSRect(x: (px - sym.size.width) / 2, y: (px - sym.size.height) / 2, width: sym.size.width, height: sym.size.height))
NSGraphicsContext.restoreGraphicsState()
try! rep.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: CommandLine.arguments[1]))
