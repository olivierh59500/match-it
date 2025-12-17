package game

import (
    "encoding/json"
    "os"
    "sort"
)

type ScoreEntry struct {
    Name  string `json:"name"`
    Score int    `json:"score"`
}

type Highscores struct {
    Entries []ScoreEntry `json:"entries"`
}

func loadHighscores(path string) (*Highscores, error) {
    f, err := os.Open(path)
    if err != nil {
        if os.IsNotExist(err) {
            return &Highscores{Entries: make([]ScoreEntry, 0, 10)}, nil
        }
        return nil, err
    }
    defer f.Close()
    var hs Highscores
    if err := json.NewDecoder(f).Decode(&hs); err != nil {
        // If decode fails, return empty table
        return &Highscores{Entries: make([]ScoreEntry, 0, 10)}, nil
    }
    return &hs, nil
}

func saveHighscores(path string, hs *Highscores) error {
    tmp := path + ".tmp"
    f, err := os.Create(tmp)
    if err != nil { return err }
    enc := json.NewEncoder(f)
    enc.SetIndent("", "  ")
    if err := enc.Encode(hs); err != nil { f.Close(); return err }
    f.Close()
    return os.Rename(tmp, path)
}

func (hs *Highscores) submit(e ScoreEntry) {
    hs.Entries = append(hs.Entries, e)
    sort.Slice(hs.Entries, func(i, j int) bool { return hs.Entries[i].Score > hs.Entries[j].Score })
    if len(hs.Entries) > 10 {
        hs.Entries = hs.Entries[:10]
    }
}

