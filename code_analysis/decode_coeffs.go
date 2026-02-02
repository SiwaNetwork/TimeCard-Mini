// +build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

func hexToDouble(hex string) float64 {
	var b [8]byte
	for i := 0; i < 8 && i*2+2 <= len(hex); i++ {
		var x byte
		fmt.Sscanf(hex[i*2:i*2+2], "%02x", &x)
		b[i] = x
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b[:]))
}

func main() {
	pairs := []struct {
		addr string
		hex  string
	}{
		{"770b7e0", "000000000000f0bf"},
		{"770b8a0", "000000000000e03f"},
		{"770b8a8", "15b7310afe06e33f"},
		{"770b8b0", "cc3b7f669ea0e63f"},
		{"770b8b8", "acd35a999fe8ea3f"},
	}
	fmt.Println("ДЕКОДИРОВАНИЕ КОЭФФИЦИЕНТОВ (DefaultAlgoCoefficients)")
	for _, p := range pairs {
		fmt.Printf("  %s: %g\n", p.addr, hexToDouble(p.hex))
	}
	fmt.Println("\nВозможные Kp, Ki, Kd (offset 0, 8, 16 от 770b8a0):")
	fmt.Printf("  Kp = %g\n", hexToDouble("000000000000e03f"))
	fmt.Printf("  Ki = %g\n", hexToDouble("15b7310afe06e33f"))
	fmt.Printf("  Kd = %g\n", hexToDouble("cc3b7f669ea0e63f"))
	fmt.Println("\nСледующие 3 значения (для D-массива?):")
	fmt.Printf("  [0] = %g\n", hexToDouble("cc3b7f669ea0e63f"))
	fmt.Printf("  [1] = %g\n", hexToDouble("acd35a999fe8ea3f"))
}
