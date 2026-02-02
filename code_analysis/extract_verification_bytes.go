// +build ignore

// extract_verification_bytes извлекает байты VerificationReqOne/Two/Three и VerificationRespOne/Two/Three
// из бинарника timebeat (ELF). Запуск на Linux/WSL: go run extract_verification_bytes.go [путь/к/timebeat]
// Символы в бинарнике: .../eth_sw_KSZ9567S.VerificationReqOne, VerificationRespOne и т.д.
package main

import (
	"debug/elf"
	"fmt"
	"os"
)

func main() {
	path := "timebeat-extracted/usr/share/timebeat/bin/timebeat"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := elf.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()

	syms := []string{
		"github.com/lasselj/timebeat/beater/clocksync/clients/vendors/timebeat/eth_sw/eth_sw_KSZ9567S.VerificationReqOne",
		"VerificationReqOne",
		"VerificationReqTwo",
		"VerificationReqThree",
		"VerificationRespOne",
		"VerificationRespTwo",
		"VerificationRespThree",
	}
	symbols, _ := f.Symbols()
	for _, sym := range symbols {
		for _, name := range syms {
			if sym.Name == name || len(sym.Name) > len(name) && sym.Name[len(sym.Name)-len(name):] == name {
				sec := f.SectionByIndex(uint16(sym.Section))
				if sec == nil {
					continue
				}
				data, err := sec.Data()
				if err != nil {
					continue
				}
				off := int(sym.Value - sec.Addr)
				if off < 0 || off >= len(data) {
					continue
				}
				n := int(sym.Size)
				if n <= 0 {
					n = 4
				}
				if off+n > len(data) {
					n = len(data) - off
				}
				fmt.Printf("%s: % x\n", sym.Name, data[off:off+n])
				break
			}
		}
	}
}
