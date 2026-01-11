package ui

import "github.com/mattn/go-runewidth"

// TruncateString truncates a string to the given visual width.
// It handles multi-byte characters and full-width characters correctly.
// If the string is truncated, "..." is appended (and accounted for in width).
// Pre-condition: width should be at least 3.
func TruncateString(s string, width int) string {
    if width < 3 {
        // Fallback for very small width
        runes := []rune(s)
        if len(runes) > width {
            return string(runes[:width])
        }
        return s
    }

    if runewidth.StringWidth(s) <= width {
        return s
    }

    targetWidth := width - 3
    
    currentWidth := 0
    runes := []rune(s)
    for i, r := range runes {
        w := runewidth.RuneWidth(r)
        if currentWidth + w > targetWidth {
            return string(runes[:i]) + "..."
        }
        currentWidth += w
    }
    
    return s
}

// TruncateStart truncates the start of the string if it exceeds width.
// "..." + suffix
func TruncateStart(s string, width int) string {
    if width < 3 {
        runes := []rune(s)
        if len(runes) > width {
            return string(runes[len(runes)-width:])
        }
        return s
    }

    if runewidth.StringWidth(s) <= width {
        return s
    }
    
    targetWidth := width - 3
    runes := []rune(s)
    
    // Calculate total width first
    totalWidth := 0
    for _, r := range runes {
        totalWidth += runewidth.RuneWidth(r)
    }
    
    // Scan from end
    currentWidth := 0
    for i := len(runes) - 1; i >= 0; i-- {
        w := runewidth.RuneWidth(runes[i])
        if currentWidth + w > targetWidth {
            return "..." + string(runes[i+1:])
        }
        currentWidth += w
    }
    
    return "..." + s
}
