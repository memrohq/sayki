// Renders the DMG window background: swift dmg-background.swift out.png scale
// Layout matches dmg-settings.py: 660x400 window, icons centred at x=170 and x=490, y=150.
// Finder's title bar and path bar take ~60pt of the 400, so keep content above y≈320.
import AppKit

let w = 660.0, h = 400.0
let scale = Double(CommandLine.arguments[2])!
let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(w * scale), pixelsHigh: Int(h * scale), bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
rep.size = NSSize(width: w, height: h) // draw in points; the rep supplies the pixels
NSGraphicsContext.saveGraphicsState()
let ctx = NSGraphicsContext(bitmapImageRep: rep)!
NSGraphicsContext.current = ctx
// Flip so y grows downward, like Finder's icon coordinates.
ctx.cgContext.translateBy(x: 0, y: h)
ctx.cgContext.scaleBy(x: 1, y: -1)

let accentA = NSColor(red: 1, green: 0.42, blue: 0.36, alpha: 1)
let accentB = NSColor(red: 0.85, green: 0.2, blue: 0.32, alpha: 1)

// Soft warm-white background with a faint accent glow behind the arrow.
NSGradient(starting: NSColor(white: 0.985, alpha: 1), ending: NSColor(red: 0.95, green: 0.94, blue: 0.94, alpha: 1))!
    .draw(in: NSRect(x: 0, y: 0, width: w, height: h), angle: 90)
NSGradient(colors: [accentA.withAlphaComponent(0.10), accentA.withAlphaComponent(0)])!
    .draw(fromCenter: NSPoint(x: w / 2, y: 150), radius: 0, toCenter: NSPoint(x: w / 2, y: 150), radius: 220, options: [])

// Arrow from the app icon to Applications: a rounded shaft and chevron head.
let arrow = NSBezierPath()
arrow.move(to: NSPoint(x: 262, y: 150))
arrow.line(to: NSPoint(x: 392, y: 150))
arrow.move(to: NSPoint(x: 376, y: 136))
arrow.line(to: NSPoint(x: 396, y: 150))
arrow.line(to: NSPoint(x: 376, y: 164))
arrow.lineWidth = 6
arrow.lineCapStyle = .round
arrow.lineJoinStyle = .round
NSGraphicsContext.saveGraphicsState()
ctx.cgContext.addPath(arrow.cgPath)
ctx.cgContext.setLineWidth(6)
ctx.cgContext.setLineCap(.round)
ctx.cgContext.setLineJoin(.round)
ctx.cgContext.replacePathWithStrokedPath()
ctx.cgContext.clip()
NSGradient(starting: accentA, ending: accentB)!.draw(in: NSRect(x: 255, y: 130, width: 150, height: 40), angle: 0)
NSGraphicsContext.restoreGraphicsState()

// Caption under the icons (Finder draws the icon labels itself at ~y=232).
func centred(_ text: String, y: Double, size: CGFloat, weight: NSFont.Weight, color: NSColor) {
    let attrs: [NSAttributedString.Key: Any] = [.font: NSFont.systemFont(ofSize: size, weight: weight), .foregroundColor: color]
    let s = NSAttributedString(string: text, attributes: attrs)
    let sz = s.size()
    // Text draws upright in a flipped context only via a flipped NSGraphicsContext.
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = NSGraphicsContext(cgContext: ctx.cgContext, flipped: true)
    s.draw(at: NSPoint(x: (w - sz.width) / 2, y: y))
    NSGraphicsContext.restoreGraphicsState()
}
centred("Drag Sayki into Applications", y: 262, size: 15, weight: .semibold, color: NSColor(white: 0.18, alpha: 1))
centred("Then open it from Applications or Spotlight", y: 286, size: 12, weight: .regular, color: NSColor(white: 0.45, alpha: 1))

NSGraphicsContext.restoreGraphicsState()
try! rep.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: CommandLine.arguments[1]))
