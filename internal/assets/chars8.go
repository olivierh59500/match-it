package assets

import (
    "bufio"
    "bytes"
    "fmt"
    "os"
    "strconv"
    "strings"
)

// LoadChars8FromAsm parses the 'chars8:' DC.W table from the original assembly file
// and returns the raw byte table arranged exactly like in memory: 8 rows of 32 bytes per block,
// with one byte per column per row. Each DC.W contributes its high byte only (big-endian words),
// matching how textaus3 accesses chars8 via move.b (A2) with a stride of 32 bytes per row.
func LoadChars8FromAsm(path string) ([]byte, error) {
    raw, err := os.ReadFile(path)
    if err != nil { return nil, err }
    s := string(raw)
    pos := strings.Index(s, "chars8:")
    if pos < 0 { return nil, fmt.Errorf("chars8 label not found") }
    // Start parsing right after the label line to avoid off-by-one with CRLF
    sec := s[pos+len("chars8:"):]
    lines := bufio.NewScanner(strings.NewReader(sec))
    var out bytes.Buffer
    for lines.Scan() {
        line := lines.Text()
        // stop at next label (e.g., 'kasten:')
        trim := strings.TrimSpace(line)
        if strings.HasSuffix(trim, ":") {
            break
        }
        // remove comment
        if i := strings.Index(trim, ";"); i >= 0 {
            trim = trim[:i]
        }
        trim = strings.TrimSpace(trim)
        if trim == "" { continue }
        // accept only DC.W rows
        up := strings.ToUpper(trim)
        if !strings.HasPrefix(up, "DC.W") && !strings.HasPrefix(up, "DCB.W") && !strings.HasPrefix(up, "DC.B") {
            continue
        }
        // Handle DCB.W (replicate) and DC.W (words) or DC.B (bytes)
        if strings.HasPrefix(up, "DCB.W") {
            // DCB.W count,value
            rest := strings.TrimSpace(strings.TrimPrefix(up, "DCB.W"))
            parts := strings.Split(rest, ",")
            if len(parts) == 2 {
                cntStr := strings.TrimSpace(parts[0])
                valStr := strings.TrimSpace(parts[1])
                cnt, _ := strconv.Atoi(cntStr)
                var val int64
                if strings.HasPrefix(valStr, "$") { val, _ = strconv.ParseInt("0x"+valStr[1:], 0, 16) } else { val, _ = strconv.ParseInt(valStr, 10, 16) }
                hb := byte(uint16(val) >> 8)
                for i := 0; i < cnt; i++ { out.WriteByte(hb) }
            }
            continue
        }
        if strings.HasPrefix(up, "DC.B") {
            rest := strings.TrimSpace(strings.TrimPrefix(up, "DC.B"))
            parts := strings.Split(rest, ",")
            for _, p := range parts {
                t := strings.TrimSpace(p)
                if t == "" { continue }
                var val int64
                if strings.HasPrefix(t, "$") { val, _ = strconv.ParseInt("0x"+t[1:], 0, 8) } else { val, _ = strconv.ParseInt(t, 10, 8) }
                out.WriteByte(byte(val))
            }
            continue
        }
        // DC.W: take only the high byte per word
        rest := strings.TrimSpace(strings.TrimPrefix(up, "DC.W"))
        parts := strings.Split(rest, ",")
        for _, p := range parts {
            t := strings.TrimSpace(p)
            if t == "" { continue }
            var val int64
            if strings.HasPrefix(t, "$") { val, _ = strconv.ParseInt("0x"+t[1:], 0, 16) } else { val, _ = strconv.ParseInt(t, 10, 16) }
            out.WriteByte(byte(uint16(val) >> 8))
        }
    }
    return out.Bytes(), nil
}
