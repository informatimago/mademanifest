package main

import (
    "io"
    "os"
    "path/filepath"
)

func main() {
    src := filepath.Join("..", "..", "..", "golden", "GOLDEN_TEST_CASE_V1.json")
    dst := filepath.Join("..", "..", "..", "data", "GOLDEN_TEST_CASE_V1.json")
    os.MkdirAll(filepath.Dir(dst), 0755)
    fsrc, _ := os.Open(src)
    defer fsrc.Close()
    fdst, _ := os.Create(dst)
    defer fdst.Close()
    io.Copy(fdst, fsrc)
}
