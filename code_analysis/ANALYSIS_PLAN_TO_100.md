# 📋 План анализа до 100% готовности

**Текущая готовность:** 85%  
**Цель:** 100%  
**Осталось:** 15%

---

## 🎯 Этап 1: Коэффициенты алгоритмов (85% → 90%) ⚡ В ПРОЦЕССЕ

**Текущий прогресс:** 85% → **88%** ✅

### 1.1 PID коэффициенты

**Цель:** Найти Kp, Ki, Kd для PID алгоритма

**Шаги:**

1. **Найти структуру CoefficientStore**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i coefficient
objdump -T /usr/share/shiwatime/bin/shiwatime | grep -i coefficient
```

2. **Дизассемблировать CalculateNewFrequency**
```bash
# Найти адрес
nm -D /usr/share/shiwatime/bin/shiwatime | grep CalculateNewFrequency

# Дизассемблировать
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 200 "CalculateNewFrequency"
```

3. **Найти константы с плавающей точкой**
```bash
# В бинарнике
strings /usr/share/shiwatime/bin/shiwatime | grep -E "^[0-9]+\.[0-9]+$"

# В ассемблере (поиск загрузок констант)
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -E "ldr.*#[0-9]+\.[0-9]+" | head -20
```

4. **Проанализировать enforceAdjustmentLimit**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep enforceAdjustmentLimit
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 100 "enforceAdjustmentLimit"
```

**Ожидаемый результат:**
- Значения Kp, Ki, Kd (float64)
- Лимиты коррекции (min/max)
- Параметры фильтрации

**Скрипт:** `extract_pid_coefficients.py`

---

### 1.2 PI коэффициенты

**Аналогично PID**, но искать только Kp и Ki

**Скрипт:** `extract_pi_coefficients.py`

---

### 1.3 LinReg параметры ✅ НАЙДЕНО

**Цель:** Найти параметры линейной регрессии

**✅ Что уже найдено:**

1. **Размер окна регрессии:** ✅ **64 элемента** (0x40)
   ```assembly
   41c6e4c: f101005f  cmp x2, #0x40  // Проверка на 64
   ```

2. **regress проанализирован:**
   - Адрес: 0x41c6e30
   - Алгоритм: линейная регрессия по 6 интервалам (2-6)
   - Формула: `slope = (Σxy - Σx·Σy/n) / (Σx² - (Σx)²/n)`

3. **Структура linreg_servo:**
   - `0x638` - массив данных (64 элемента)
   - `0x1536` - референсное время
   - `0x1544` - референсное смещение
   - `0x1560` - счетчик элементов
   - `0x1568` - индекс текущего элемента
   - `0x1752` - размер окна
   - `0x1760` - частота коррекции
   - `0x1776` - коэффициент частоты

4. **Параметры:**
   - Интервалы: 2, 3, 4, 5, 6 (5 интервалов)
   - Порог усреднения: 10 (0xa)
   - Константы из памяти (504b000): offset 416, 872, 1632

**Шаги для завершения:**

1. **Проанализировать update_reference**
```bash
shiwa@grandmini:~ $ nm -D /usr/share/shiwatime/bin/shiwatime | grep enforceAdjustmentLimit

objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 100 "enforceAdjustmentLimit"      
00000000041c8cc0 T github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit
 41c8978:       940000d2        bl      41c8cc0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base>
 41c897c:       36000061        tbz     w1, #0, 41c8988 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).CalculateNewFrequency@@Base+0x1c8>
 41c8980:       f940e7e8        ldr     x8, [sp, #456]
 41c8984:       14000007        b       41c89a0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).CalculateNewFrequency@@Base+0x1e0>
 41c8988:       f940e7e8        ldr     x8, [sp, #456]
 41c898c:       fd400d03        ldr     d3, [x8, #24]
 41c8990:       fd4047e4        ldr     d4, [sp, #136]
 41c8994:       fd4043e5        ldr     d5, [sp, #128]
 41c8998:       1f440ca3        fmadd   d3, d5, d4, d3
 41c899c:       fd000d03        str     d3, [x8, #24]
 41c89a0:       f9000500        str     x0, [x8, #8]
 41c89a4:       39420109        ldrb    w9, [x8, #128]
 41c89a8:       360007a9        tbz     w9, #0, 41c8a9c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).CalculateNewFrequency@@Base+0x2dc>
 41c89ac:       f90037e0        str     x0, [sp, #104]
 41c89b0:       910563f4        add     x20, sp, #0x158
 41c89b4:       1000009b        adr     x27, 41c89c4 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).CalculateNewFrequency@@Base+0x204>
 41c89b8:       a93eeffd        stp     x29, x27, [sp, #-24]
 41c89bc:       d10063fd        sub     x29, sp, #0x18
 41c89c0:       9778f8ba        bl      2006ca8 <runtime.duffzero@@Base+0xe8>
 41c89c4:       d10023fd        sub     x29, sp, #0x8
 41c89c8:       f940ebe0        ldr     x0, [sp, #464]
 41c89cc:       977755ad        bl      1f9e080 <runtime.convT64@@Base>
 41c89d0:       f0008ae1        adrp    x1, 5327000 <type:dDN11xKD@@Base+0x20>
 41c89d4:       913e8021        add     x1, x1, #0xfa0
 41c89d8:       f900afe1        str     x1, [sp, #344]
 41c89dc:       f900b3e0        str     x0, [sp, #352]
 41c89e0:       fd4027e0        ldr     d0, [sp, #72]
 41c89e4:       9e660000        fmov    x0, d0
 41c89e8:       977755a6        bl      1f9e080 <runtime.convT64@@Base>
 41c89ec:       900088c1        adrp    x1, 52e0000 <type:xuIQmDSe@@Base+0x20>
 41c89f0:       910e8021        add     x1, x1, #0x3a0
 41c89f4:       f900b7e1        str     x1, [sp, #360]
 41c89f8:       f900bbe0        str     x0, [sp, #368]
 41c89fc:       fd402fe0        ldr     d0, [sp, #88]
 41c8a00:       9e660000        fmov    x0, d0
 41c8a04:       9777559f        bl      1f9e080 <runtime.convT64@@Base>
 41c8a08:       900088c1        adrp    x1, 52e0000 <type:xuIQmDSe@@Base+0x20>
 41c8a0c:       910e8021        add     x1, x1, #0x3a0
 41c8a10:       f900bfe1        str     x1, [sp, #376]
 41c8a14:       f900c3e0        str     x0, [sp, #384]
 41c8a18:       fd4033e0        ldr     d0, [sp, #96]
 41c8a1c:       9e660000        fmov    x0, d0
 41c8a20:       97775598        bl      1f9e080 <runtime.convT64@@Base>
 41c8a24:       900088c1        adrp    x1, 52e0000 <type:xuIQmDSe@@Base+0x20>
 41c8a28:       910e8021        add     x1, x1, #0x3a0
 41c8a2c:       f900c7e1        str     x1, [sp, #392]
 41c8a30:       f900cbe0        str     x0, [sp, #400]
 41c8a34:       f94023e0        ldr     x0, [sp, #64]
 41c8a38:       97775592        bl      1f9e080 <runtime.convT64@@Base>
 41c8a3c:       f0008ae1        adrp    x1, 5327000 <type:dDN11xKD@@Base+0x20>
 41c8a40:       913e8021        add     x1, x1, #0xfa0
 41c8a44:       f900cfe1        str     x1, [sp, #408]
 41c8a48:       f900d3e0        str     x0, [sp, #416]
 41c8a4c:       f94037e0        ldr     x0, [sp, #104]
 41c8a50:       9777558c        bl      1f9e080 <runtime.convT64@@Base>
 41c8a54:       f0008ae1        adrp    x1, 5327000 <type:dDN11xKD@@Base+0x20>
 41c8a58:       913e8021        add     x1, x1, #0xfa0
 41c8a5c:       f900d7e1        str     x1, [sp, #424]
 41c8a60:       f900dbe0        str     x0, [sp, #432]
 41c8a64:       f001baa1        adrp    x1, 791f000 <k8s.io/api/policy/v1.map_PodDisruptionBudget@@Base>
 41c8a68:       9121a021        add     x1, x1, #0x868
 41c8a6c:       f9400021        ldr     x1, [x1]
 41c8a70:       d000c560        adrp    x0, 5a76000 <go:itab.*net.AddrError,error@@Base>  
 41c8a74:       91160000        add     x0, x0, #0x580
 41c8a78:       d0003f02        adrp    x2, 49aa000 <_IO_stdin_used@@Base+0x1fcc80>       
 41c8a7c:       913b8842        add     x2, x2, #0xee2
 41c8a80:       d28005e3        mov     x3, #0x2f                       // #47
 41c8a84:       910563e4        add     x4, sp, #0x158
 41c8a88:       b27f07e5        orr     x5, xzr, #0x6
 41c8a8c:       aa0503e6        mov     x6, x5
 41c8a90:       977b52b8        bl      209d570 <fmt.Fprintf@@Base>
 41c8a94:       f94037e0        ldr     x0, [sp, #104]
 41c8a98:       f940e7e8        ldr     x8, [sp, #456]
 41c8a9c:       f940ebe9        ldr     x9, [sp, #464]
 41c8aa0:       f9000109        str     x9, [x8]
 41c8aa4:       910403f4        add     x20, sp, #0x100
 41c8aa8:       1000009b        adr     x27, 41c8ab8 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).CalculateNewFrequency@@Base+0x2f8>
 41c8aac:       a93eeffd        stp     x29, x27, [sp, #-24]
 41c8ab0:       d10063fd        sub     x29, sp, #0x18
 41c8ab4:       9778f87e        bl      2006cac <runtime.duffzero@@Base+0xec>
 41c8ab8:       d10023fd        sub     x29, sp, #0x8
 41c8abc:       f900abff        str     xzr, [sp, #336]
 41c8ac0:       f9402509        ldr     x9, [x8, #72]
 41c8ac4:       f940290a        ldr     x10, [x8, #80]
 41c8ac8:       f90083e9        str     x9, [sp, #256]
 41c8acc:       f90087ea        str     x10, [sp, #264]
 41c8ad0:       f9008be0        str     x0, [sp, #272]
 41c8ad4:       fd4027e3        ldr     d3, [sp, #72]
 41c8ad8:       9e780069        fcvtzs  x9, d3
 41c8adc:       9e620123        scvtf   d3, x9
 41c8ae0:       fd008fe3        str     d3, [sp, #280]
 41c8ae4:       fd402fe3        ldr     d3, [sp, #88]
 41c8ae8:       9e780069        fcvtzs  x9, d3
 41c8aec:       9e620123        scvtf   d3, x9
 41c8af0:       fd0093e3        str     d3, [sp, #288]
 41c8af4:       fd4033e3        ldr     d3, [sp, #96]
 41c8af8:       9e780069        fcvtzs  x9, d3
 41c8afc:       9e620123        scvtf   d3, x9
 41c8b00:       fd0097e3        str     d3, [sp, #296]
 41c8b04:       39410108        ldrb    w8, [x8, #64]
 41c8b08:       390523e8        strb    w8, [sp, #328]
--
00000000041c8cc0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base>:
 41c8cc0:       f9400b90        ldr     x16, [x28, #16]
 41c8cc4:       eb3063ff        cmp     sp, x16
 41c8cc8:       54000589        b.ls    41c8d78 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0xb8>  // b.plast
 41c8ccc:       f81d0ffe        str     x30, [sp, #-48]!
 41c8cd0:       f81f83fd        stur    x29, [sp, #-8]
 41c8cd4:       d10023fd        sub     x29, sp, #0x8
 41c8cd8:       f9403802        ldr     x2, [x0, #112]
 41c8cdc:       b4000202        cbz     x2, 41c8d1c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x5c>
 41c8ce0:       f9403003        ldr     x3, [x0, #96]
 41c8ce4:       b5000063        cbnz    x3, 41c8cf0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x30>
 41c8ce8:       b5000222        cbnz    x2, 41c8d2c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x6c>
 41c8cec:       1400000c        b       41c8d1c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x5c>
 41c8cf0:       f90023e1        str     x1, [sp, #64]
 41c8cf4:       9e620060        scvtf   d0, x3
 41c8cf8:       fd0007e0        str     d0, [sp, #8]
 41c8cfc:       9e620040        scvtf   d0, x2
 41c8d00:       fd000be0        str     d0, [sp, #16]
 41c8d04:       977943cb        bl      2019c30 <math.archMin.abi0@@Base>
 41c8d08:       fd400fe0        ldr     d0, [sp, #24]
 41c8d0c:       9e780000        fcvtzs  x0, d0
 41c8d10:       f94023e1        ldr     x1, [sp, #64]
 41c8d14:       aa0003e2        mov     x2, x0
 41c8d18:       14000005        b       41c8d2c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x6c>
 41c8d1c:       f9403002        ldr     x2, [x0, #96]
 41c8d20:       b5000062        cbnz    x2, 41c8d2c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x6c>
 41c8d24:       f9401403        ldr     x3, [x0, #40]
 41c8d28:       f9401062        ldr     x2, [x3, #32]
 41c8d2c:       eb02003f        cmp     x1, x2
 41c8d30:       540001ac        b.gt    41c8d64 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0xa4>
 41c8d34:       cb0203e0        neg     x0, x2
 41c8d38:       eb00003f        cmp     x1, x0
 41c8d3c:       540000aa        b.ge    41c8d50 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base+0x90>  // b.tcont
 41c8d40:       b24003e1        orr     x1, xzr, #0x1
 41c8d44:       a97ffbfd        ldp     x29, x30, [sp, #-8]
 41c8d48:       9100c3ff        add     sp, sp, #0x30
 41c8d4c:       d65f03c0        ret
 41c8d50:       aa0103e0        mov     x0, x1
 41c8d54:       aa1f03e1        mov     x1, xzr
 41c8d58:       a97ffbfd        ldp     x29, x30, [sp, #-8]
 41c8d5c:       9100c3ff        add     sp, sp, #0x30
 41c8d60:       d65f03c0        ret
 41c8d64:       aa0203e0        mov     x0, x2
 41c8d68:       b24003e1        orr     x1, xzr, #0x1
 41c8d6c:       a97ffbfd        ldp     x29, x30, [sp, #-8]
 41c8d70:       9100c3ff        add     sp, sp, #0x30
 41c8d74:       d65f03c0        ret
 41c8d78:       f90007e0        str     x0, [sp, #8]
 41c8d7c:       f9000be1        str     x1, [sp, #16]
 41c8d80:       aa1e03e3        mov     x3, x30
 41c8d84:       9778ed67        bl      2004320 <runtime.morestack_noctxt.abi0@@Base>     
 41c8d88:       f94007e0        ldr     x0, [sp, #8]
 41c8d8c:       f9400be1        ldr     x1, [sp, #16]
 41c8d90:       17ffffcc        b       41c8cc0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).enforceAdjustmentLimit@@Base>
        ...

00000000041c8da0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base>:
 41c8da0:       f9400b90        ldr     x16, [x28, #16]
 41c8da4:       eb3063ff        cmp     sp, x16
 41c8da8:       54001149        b.ls    41c8fd0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x230>  // b.plast
 41c8dac:       f81f0ffe        str     x30, [sp, #-16]!
 41c8db0:       f81f83fd        stur    x29, [sp, #-8]
 41c8db4:       d10023fd        sub     x29, sp, #0x8
 41c8db8:       fd403801        ldr     d1, [x0, #112]
 41c8dbc:       f9403c04        ldr     x4, [x0, #120]
 41c8dc0:       b50000e4        cbnz    x4, 41c8ddc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x3c>
 41c8dc4:       f9001c01        str     x1, [x0, #56]
 41c8dc8:       f9002402        str     x2, [x0, #72]
 41c8dcc:       f900007f        str     xzr, [x3]
 41c8dd0:       b24003e1        orr     x1, xzr, #0x1
 41c8dd4:       f9003c01        str     x1, [x0, #120]
 41c8dd8:       14000077        b       41c8fb4 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x214>
 41c8ddc:       f100049f        cmp     x4, #0x1
 41c8de0:       54000a61        b.ne    41c8f2c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x18c>  // b.any
 41c8de4:       f9002001        str     x1, [x0, #64]
 41c8de8:       f9002802        str     x2, [x0, #80]
 41c8dec:       f9402402        ldr     x2, [x0, #72]
 41c8df0:       f9402804        ldr     x4, [x0, #80]
 41c8df4:       eb02009f        cmp     x4, x2
 41c8df8:       54000088        b.hi    41c8e08 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x68>  // b.pmore
 41c8dfc:       f900007f        str     xzr, [x3]
 41c8e00:       f9003c1f        str     xzr, [x0, #120]
 41c8e04:       1400006c        b       41c8fb4 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x214>
 41c8e08:       d341fc45        lsr     x5, x2, #1
 41c8e0c:       d294b306        mov     x6, #0xa598                     // #42392
 41c8e10:       f2a6d686        movk    x6, #0x36b4, lsl #16
 41c8e14:       f2cbe826        movk    x6, #0x5f41, lsl #32
 41c8e18:       f2f12e06        movk    x6, #0x8970, lsl #48
 41c8e1c:       9bc57cc5        umulh   x5, x6, x5
 41c8e20:       cb457085        sub     x5, x4, x5, lsr #28
 41c8e24:       9e6300a2        ucvtf   d2, x5
 41c8e28:       f000741b        adrp    x27, 504b000 <_IO_stdin_used@@Base+0x89dc80>      
 41c8e2c:       fd40b363        ldr     d3, [x27, #352]
 41c8e30:       1f430842        fmadd   d2, d2, d3, d2
 41c8e34:       fd403403        ldr     d3, [x0, #104]
 41c8e38:       f000741b        adrp    x27, 504b000 <_IO_stdin_used@@Base+0x89dc80>      
 41c8e3c:       fd40cf64        ldr     d4, [x27, #408]
 41c8e40:       1e631883        fdiv    d3, d4, d3
 41c8e44:       f000741b        adrp    x27, 504b000 <_IO_stdin_used@@Base+0x89dc80>      
 41c8e48:       fd428f64        ldr     d4, [x27, #1304]
 41c8e4c:       1e632080        fcmp    d4, d3
 41c8e50:       54000065        b.pl    41c8e5c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0xbc>  // b.nfrst
 41c8e54:       f000741b        adrp    x27, 504b000 <_IO_stdin_used@@Base+0x89dc80>      
 41c8e58:       fd428f63        ldr     d3, [x27, #1304]
 41c8e5c:       1e632040        fcmp    d2, d3
 41c8e60:       54000065        b.pl    41c8e6c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0xcc>  // b.nfrst
 41c8e64:       f900007f        str     xzr, [x3]
 41c8e68:       14000053        b       41c8fb4 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x214>
 41c8e6c:       fd402c01        ldr     d1, [x0, #88]
 41c8e70:       9e780025        fcvtzs  x5, d1
 41c8e74:       d2994006        mov     x6, #0xca00                     // #51712
 41c8e78:       f2a77346        movk    x6, #0x3b9a, lsl #16
 41c8e7c:       cb0500c5        sub     x5, x6, x5
 41c8e80:       f9401c06        ldr     x6, [x0, #56]
 41c8e84:       cb060026        sub     x6, x1, x6
 41c8e88:       9b067ca5        mul     x5, x5, x6
 41c8e8c:       cb020082        sub     x2, x4, x2
 41c8e90:       b40009c2        cbz     x2, 41c8fc8 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x228>
 41c8e94:       9ac20ca2        sdiv    x2, x5, x2
 41c8e98:       9e620042        scvtf   d2, x2
 41c8e9c:       1e622821        fadd    d1, d1, d2
 41c8ea0:       fd002c01        str     d1, [x0, #88]
 41c8ea4:       fd400002        ldr     d2, [x0]
 41c8ea8:       1e614043        fneg    d3, d2
 41c8eac:       1e632020        fcmp    d1, d3
 41c8eb0:       54000065        b.pl    41c8ebc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x11c>  // b.nfrst
 41c8eb4:       fd002c03        str     d3, [x0, #88]
 41c8eb8:       14000004        b       41c8ec8 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x128>
 41c8ebc:       1e612040        fcmp    d2, d1
 41c8ec0:       54000045        b.pl    41c8ec8 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x128>  // b.nfrst
 41c8ec4:       fd002c02        str     d2, [x0, #88]
 41c8ec8:       f9400c02        ldr     x2, [x0, #24]
 41c8ecc:       b4000102        cbz     x2, 41c8eec <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x14c>
 41c8ed0:       fd400801        ldr     d1, [x0, #16]
 41c8ed4:       1e602028        fcmp    d1, #0.0
 41c8ed8:       540000a0        b.eq    41c8eec <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x14c>  // b.none
 41c8edc:       9e620022        scvtf   d2, x1
 41c8ee0:       1e60c042        fabs    d2, d2
 41c8ee4:       1e622020        fcmp    d1, d2
 41c8ee8:       54000104        b.mi    41c8f08 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x168>  // b.first
 41c8eec:       fd400401        ldr     d1, [x0, #8]
 41c8ef0:       1e602028        fcmp    d1, #0.0
 41c8ef4:       54000100        b.eq    41c8f14 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x174>  // b.none
 41c8ef8:       9e620022        scvtf   d2, x1
 41c8efc:       1e60c042        fabs    d2, d2
 41c8f00:       1e622020        fcmp    d1, d2
 41c8f04:       54000085        b.pl    41c8f14 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x174>  // b.nfrst
 41c8f08:       b24003e1        orr     x1, xzr, #0x1
 41c8f0c:       f9000061        str     x1, [x3]
 41c8f10:       14000003        b       41c8f1c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*pi_servo).pi_sample@@Base+0x17c>
 41c8f14:       b27f03e1        orr     x1, xzr, #0x2
 41c8f18:       f9000061        str     x1, [x3]
 41c8f1c:       fd402c01        ldr     d1, [x0, #88]
 41c8f20:       b27f03e1        orr     x1, xzr, #0x2
shiwa@grandmini:~ $ python3 extract_coefficients_v2.py > ~/coeffs_final_v2.txt
shiwa@grandmini:~ $ objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 100 "update_reference"
00000000041c6cc0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).update_reference@@Base>:
 41c6cc0:       f9431802        ldr     x2, [x0, #1584]
 41c6cc4:       b40002e2        cbz     x2, 41c6d20 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).update_reference@@Base+0x60>
 41c6cc8:       cb020022        sub     x2, x1, x2
 41c6ccc:       9e620040        scvtf   d0, x2
 41c6cd0:       fd437001        ldr     d1, [x0, #1760]
 41c6cd4:       b000743b        adrp    x27, 504b000 <_IO_stdin_used@@Base+0x89dc80>
 41c6cd8:       fd433362        ldr     d2, [x27, #1632]
 41c6cdc:       1e621821        fdiv    d1, d1, d2
 41c6ce0:       1e6e1002        fmov    d2, #1.000000000000000000e+00
 41c6ce4:       1e622821        fadd    d1, d1, d2
 41c6ce8:       1e611801        fdiv    d1, d0, d1
 41c6cec:       fd431402        ldr     d2, [x0, #1576]
 41c6cf0:       1e612841        fadd    d1, d2, d1
 41c6cf4:       1e613822        fsub    d2, d1, d1
 41c6cf8:       fd031402        str     d2, [x0, #1576]
 41c6cfc:       9e780023        fcvtzs  x3, d1
 41c6d00:       f9430004        ldr     x4, [x0, #1536]
 41c6d04:       8b030084        add     x4, x4, x3
 41c6d08:       f9030004        str     x4, [x0, #1536]
 41c6d0c:       f9430404        ldr     x4, [x0, #1544]
 41c6d10:       8b020082        add     x2, x4, x2
 41c6d14:       f9030402        str     x2, [x0, #1544]
 41c6d18:       b27f03e2        orr     x2, xzr, #0x2
 41c6d1c:       1400000e        b       41c6d54 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).update_reference@@Base+0x94>
 41c6d20:       f9031801        str     x1, [x0, #1584]
 41c6d24:       d65f03c0        ret
 41c6d28:       9118e004        add     x4, x0, #0x638
 41c6d2c:       d1000845        sub     x5, x2, #0x2
 41c6d30:       8b051486        add     x6, x4, x5, lsl #5
 41c6d34:       d37be8a5        lsl     x5, x5, #5
 41c6d38:       fd4004c1        ldr     d1, [x6, #8]
 41c6d3c:       9e620062        scvtf   d2, x3
 41c6d40:       fc656883        ldr     d3, [x4, x5]
 41c6d44:       1f628062        fnmsub  d2, d3, d2, d0
 41c6d48:       1e612841        fadd    d1, d2, d1
 41c6d4c:       fd0004c1        str     d1, [x6, #8]
 41c6d50:       91000442        add     x2, x2, #0x1
 41c6d54:       f100185f        cmp     x2, #0x6
 41c6d58:       54fffe8d        b.le    41c6d28 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).update_reference@@Base+0x68>
 41c6d5c:       17fffff1        b       41c6d20 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).update_reference@@Base+0x60>

00000000041c6d60 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).add_sample@@Base>:
 41c6d60:       f9400b90        ldr     x16, [x28, #16]
 41c6d64:       eb3063ff        cmp     sp, x16
 41c6d68:       54000529        b.ls    41c6e0c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).add_sample@@Base+0xac>  // b.plast
 41c6d6c:       f81e0ffe        str     x30, [sp, #-32]!
 41c6d70:       f81f83fd        stur    x29, [sp, #-8]
 41c6d74:       d10023fd        sub     x29, sp, #0x8
 41c6d78:       f9431002        ldr     x2, [x0, #1568]
 41c6d7c:       91000442        add     x2, x2, #0x1
 41c6d80:       92401442        and     x2, x2, #0x3f
 41c6d84:       f9031002        str     x2, [x0, #1568]
 41c6d88:       cb020842        sub     x2, x2, x2, lsl #2
 41c6d8c:       cb020c02        sub     x2, x0, x2, lsl #3
 41c6d90:       f9430003        ldr     x3, [x0, #1536]
 41c6d94:       f9000043        str     x3, [x2]
 41c6d98:       f9431002        ldr     x2, [x0, #1568]
 41c6d9c:       f9430403        ldr     x3, [x0, #1544]
 41c6da0:       cb010063        sub     x3, x3, x1
 41c6da4:       f101005f        cmp     x2, #0x40
 41c6da8:       540002a2        b.cs    41c6dfc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).add_sample@@Base+0x9c>  // b.hs, b.nlast
 41c6dac:       cb020842        sub     x2, x2, x2, lsl #2
 41c6db0:       cb020c02        sub     x2, x0, x2, lsl #3
 41c6db4:       f9000443        str     x3, [x2, #8]
 41c6db8:       f9431002        ldr     x2, [x0, #1568]
 41c6dbc:       f101005f        cmp     x2, #0x40
 41c6dc0:       54000182        b.cs    41c6df0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).add_sample@@Base+0x90>  // b.hs, b.nlast
 41c6dc4:       cb020841        sub     x1, x2, x2, lsl #2
 41c6dc8:       cb010c01        sub     x1, x0, x1, lsl #3
 41c6dcc:       fd000820        str     d0, [x1, #16]
 41c6dd0:       f9430c01        ldr     x1, [x0, #1560]
 41c6dd4:       f101003f        cmp     x1, #0x40
 41c6dd8:       54000062        b.cs    41c6de4 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).add_sample@@Base+0x84>  // b.hs, b.nlast
 41c6ddc:       91000421        add     x1, x1, #0x1
 41c6de0:       f9030c01        str     x1, [x0, #1560]
 41c6de4:       a97ffbfd        ldp     x29, x30, [sp, #-8]
 41c6de8:       910083ff        add     sp, sp, #0x20
 41c6dec:       d65f03c0        ret
 41c6df0:       aa0203e0        mov     x0, x2
 41c6df4:       b27a03e1        orr     x1, xzr, #0x40
 41c6df8:       9778ff3a        bl      2006ae0 <runtime.panicIndexU@@Base>
 41c6dfc:       aa0203e0        mov     x0, x2
 41c6e00:       b27a03e1        orr     x1, xzr, #0x40
 41c6e04:       9778ff37        bl      2006ae0 <runtime.panicIndexU@@Base>
 41c6e08:       d503201f        nop
 41c6e0c:       f90007e0        str     x0, [sp, #8]
 41c6e10:       f9000be1        str     x1, [sp, #16]
 41c6e14:       fd000fe0        str     d0, [sp, #24]
 41c6e18:       aa1e03e3        mov     x3, x30
 41c6e1c:       9778f541        bl      2004320 <runtime.morestack_noctxt.abi0@@Base>     
 41c6e20:       f94007e0        ldr     x0, [sp, #8]
 41c6e24:       f9400be1        ldr     x1, [sp, #16]
 41c6e28:       fd400fe0        ldr     d0, [sp, #24]
 41c6e2c:       17ffffcd        b       41c6d60 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).add_sample@@Base>

00000000041c6e30 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base>:
 41c6e30:       f9400b90        ldr     x16, [x28, #16]
 41c6e34:       eb3063ff        cmp     sp, x16
 41c6e38:       54000d29        b.ls    41c6fdc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base+0x1ac>  // b.plast
 41c6e3c:       f81e0ffe        str     x30, [sp, #-32]!
 41c6e40:       f81f83fd        stur    x29, [sp, #-8]
 41c6e44:       d10023fd        sub     x29, sp, #0x8
 41c6e48:       f9431002        ldr     x2, [x0, #1568]
 41c6e4c:       f101005f        cmp     x2, #0x40
 41c6e50:       54000be2        b.cs    41c6fcc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base+0x19c>  // b.hs, b.nlast
 41c6e54:       cb020841        sub     x1, x2, x2, lsl #2
 41c6e58:       cb010c01        sub     x1, x0, x1, lsl #3
 41c6e5c:       f9400421        ldr     x1, [x1, #8]
 41c6e60:       f9430402        ldr     x2, [x0, #1544]
 41c6e64:       cb020021        sub     x1, x1, x2
 41c6e68:       9e620020        scvtf   d0, x1
 41c6e6c:       b27f03e1        orr     x1, xzr, #0x2
 41c6e70:       aa1f03e2        mov     x2, xzr
 41c6e74:       9e6703e1        fmov    d1, xzr
 41c6e78:       9e6703e2        fmov    d2, xzr
 41c6e7c:       9e6703e3        fmov    d3, xzr
 41c6e80:       9e6703e4        fmov    d4, xzr
 41c6e84:       9e6703e5        fmov    d5, xzr
 41c6e88:       1400000d        b       41c6ebc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base+0x8c>
 41c6e8c:       1e620826        fmul    d6, d1, d2
 41c6e90:       1e6518c6        fdiv    d6, d6, d5
 41c6e94:       1e663866        fsub    d6, d3, d6
 41c6e98:       1e610827        fmul    d7, d1, d1
 41c6e9c:       1e6518e7        fdiv    d7, d7, d5
 41c6ea0:       1e673887        fsub    d7, d4, d7
 41c6ea4:       1e6718c6        fdiv    d6, d6, d7
 41c6ea8:       fc2668a6        str     d6, [x5, x6]
 41c6eac:       1f468826        fmsub   d6, d1, d6, d2
 41c6eb0:       1e6518c6        fdiv    d6, d6, d5
 41c6eb4:       fd0004e6        str     d6, [x7, #8]
 41c6eb8:       91000421        add     x1, x1, #0x1
 41c6ebc:       f100183f        cmp     x1, #0x6
 41c6ec0:       540004ac        b.gt    41c6f54 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base+0x124>
 41c6ec4:       b24003e3        orr     x3, xzr, #0x1
 41c6ec8:       9ac12064        lsl     x4, x3, x1
 41c6ecc:       f9430c05        ldr     x5, [x0, #1560]
 41c6ed0:       eb0400bf        cmp     x5, x4
 41c6ed4:       54000403        b.cc    41c6f54 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base+0x124>  // b.lo, b.ul, b.last
 41c6ed8:       9118e005        add     x5, x0, #0x638
 41c6edc:       d1000826        sub     x6, x1, #0x2
--
 41c70bc:       97ffff01        bl      41c6cc0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).update_reference@@Base>
 41c70c0:       d503201f        nop
 41c70c4:       f94017e0        ldr     x0, [sp, #40]
 41c70c8:       f9431002        ldr     x2, [x0, #1568]
 41c70cc:       91000442        add     x2, x2, #0x1
 41c70d0:       92401442        and     x2, x2, #0x3f
 41c70d4:       f9031002        str     x2, [x0, #1568]
 41c70d8:       cb020842        sub     x2, x2, x2, lsl #2
 41c70dc:       cb020c02        sub     x2, x0, x2, lsl #3
 41c70e0:       f9430003        ldr     x3, [x0, #1536]
 41c70e4:       f9000043        str     x3, [x2]
 41c70e8:       f9431002        ldr     x2, [x0, #1568]
 41c70ec:       f9430403        ldr     x3, [x0, #1544]
 41c70f0:       f9401be4        ldr     x4, [sp, #48]
 41c70f4:       cb040063        sub     x3, x3, x4
 41c70f8:       f101005f        cmp     x2, #0x40
 41c70fc:       540014a2        b.cs    41c7390 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x300>  // b.hs, b.nlast
 41c7100:       cb020842        sub     x2, x2, x2, lsl #2
 41c7104:       cb020c02        sub     x2, x0, x2, lsl #3
 41c7108:       f9000443        str     x3, [x2, #8]
 41c710c:       f9431002        ldr     x2, [x0, #1568]
 41c7110:       f101005f        cmp     x2, #0x40
 41c7114:       54001382        b.cs    41c7384 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x2f4>  // b.hs, b.nlast
 41c7118:       cb020841        sub     x1, x2, x2, lsl #2
 41c711c:       cb010c01        sub     x1, x0, x1, lsl #3
 41c7120:       fd4023e0        ldr     d0, [sp, #64]
 41c7124:       fd000820        str     d0, [x1, #16]
 41c7128:       f9430c01        ldr     x1, [x0, #1560]
 41c712c:       f101003f        cmp     x1, #0x40
 41c7130:       54000062        b.cs    41c713c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0xac>  // b.hs, b.nlast
 41c7134:       91000421        add     x1, x1, #0x1
 41c7138:       f9030c01        str     x1, [x0, #1560]
 41c713c:       97ffff3d        bl      41c6e30 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).regress@@Base>
 41c7140:       d503201f        nop
 41c7144:       f94017e0        ldr     x0, [sp, #40]
 41c7148:       b27f03e1        orr     x1, xzr, #0x2
 41c714c:       aa1f03e2        mov     x2, xzr
 41c7150:       9e6703e0        fmov    d0, xzr
 41c7154:       14000004        b       41c7164 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0xd4>
 41c7158:       91000443        add     x3, x2, #0x1
 41c715c:       aa0103e2        mov     x2, x1
 41c7160:       aa0303e1        mov     x1, x3
 41c7164:       f100183f        cmp     x1, #0x6
 41c7168:       540003ac        b.gt    41c71dc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x14c>
 41c716c:       9118e003        add     x3, x0, #0x638
 41c7170:       d1000824        sub     x4, x1, #0x2
 41c7174:       8b041465        add     x5, x3, x4, lsl #5
 41c7178:       b50000a2        cbnz    x2, 41c718c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0xfc>
 41c717c:       d37be884        lsl     x4, x4, #5
 41c7180:       fc646861        ldr     d1, [x3, x4]
 41c7184:       1e602028        fcmp    d1, #0.0
 41c7188:       54000141        b.ne    41c71b0 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x120>  // b.any
 41c718c:       9000743b        adrp    x27, 504b000 <_IO_stdin_used@@Base+0x89dc80>      
 41c7190:       fd41b761        ldr     d1, [x27, #872]
 41c7194:       1e610802        fmul    d2, d0, d1
 41c7198:       fd4008a3        ldr     d3, [x5, #16]
 41c719c:       1e622060        fcmp    d3, d2
 41c71a0:       54000165        b.pl    41c71cc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x13c>  // b.nfrst
 41c71a4:       f9400ca3        ldr     x3, [x5, #24]
 41c71a8:       f100287f        cmp     x3, #0xa
 41c71ac:       5400008b        b.lt    41c71bc <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x12c>  // b.tstop
 41c71b0:       fd4008a0        ldr     d0, [x5, #16]
 41c71b4:       aa0103e2        mov     x2, x1
 41c71b8:       17ffffe8        b       41c7158 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0xc8>
 41c71bc:       aa0203e3        mov     x3, x2
 41c71c0:       aa0103e2        mov     x2, x1
 41c71c4:       aa0303e1        mov     x1, x3
 41c71c8:       17ffffe4        b       41c7158 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0xc8>
 41c71cc:       aa0203e3        mov     x3, x2
 41c71d0:       aa0103e2        mov     x2, x1
 41c71d4:       aa0303e1        mov     x1, x3
 41c71d8:       17ffffe0        b       41c7158 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0xc8>
 41c71dc:       f9036c02        str     x2, [x0, #1752]
 41c71e0:       f9436c02        ldr     x2, [x0, #1752]
 41c71e4:       f100085f        cmp     x2, #0x2
 41c71e8:       54000423        b.cc    41c726c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x1dc>  // b.lo, b.ul, b.last
 41c71ec:       d1000842        sub     x2, x2, #0x2
 41c71f0:       f100145f        cmp     x2, #0x5
 41c71f4:       54000c22        b.cs    41c7378 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x2e8>  // b.hs, b.nlast
 41c71f8:       9118e001        add     x1, x0, #0x638
 41c71fc:       8b021423        add     x3, x1, x2, lsl #5
 41c7200:       f9438c04        ldr     x4, [x0, #1816]
 41c7204:       b4000104        cbz     x4, 41c7224 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x194>
 41c7208:       fd438801        ldr     d1, [x0, #1808]
 41c720c:       1e602028        fcmp    d1, #0.0
 41c7210:       540000a0        b.eq    41c7224 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x194>  // b.none
 41c7214:       fd400462        ldr     d2, [x3, #8]
 41c7218:       1e60c042        fabs    d2, d2
 41c721c:       1e622020        fcmp    d1, d2
 41c7220:       54000104        b.mi    41c7240 <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x1b0>  // b.first
 41c7224:       fd438401        ldr     d1, [x0, #1800]
 41c7228:       1e602028        fcmp    d1, #0.0
 41c722c:       54000180        b.eq    41c725c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x1cc>  // b.none
 41c7230:       fd400462        ldr     d2, [x3, #8]
 41c7234:       1e60c042        fabs    d2, d2
 41c7238:       1e622020        fcmp    d1, d2
 41c723c:       54000105        b.pl    41c725c <github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*linreg_servo).linreg_sample@@Base+0x1cc>  // b.nfrst
 41c7240:       f9430404        ldr     x4, [x0, #1544]
 41c7244:       f9401be5        ldr     x5, [sp, #48]
 41c7248:       cb050084        sub     x4, x4, x5
 41c724c:       f9030404        str     x4, [x0, #1544]
shiwa@grandmini:~ $ 
```

2. **Найти пороги качества**
```bash
# Пороги в структуре (offset 1800, 1808)
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -B 5 -A 5 "1800\|1808"
```

**Ожидаемый результат:**
- ✅ Размер окна регрессии: 64 элемента
- ✅ Параметры матричных операций: найдены
- ⚠️ Пороги качества регрессии: нужно извлечь значения

**Скрипт:** `extract_linreg_parameters.py` (можно создать на основе найденного)

---

## 🎯 Этап 2: PTP/NTP детали (60% → 80%)

### 2.1 PTP модуль

**Цель:** Полный анализ всех 44 функций PTP

**Шаги:**

1. **Найти все PTP функции**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i ptp | wc -l
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i ptp | head -50
```

2. **Проанализировать master election**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i election
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 200 "HoldMasterClockElection"
```

3. **Найти PTP Squared расширения**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i squared
strings /usr/share/shiwatime/bin/shiwatime | grep -i squared
```

4. **Проанализировать PTP сообщения**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -E "(SYNC|FOLLOW_UP|DELAY_REQ|DELAY_RESP)"
```

**Ожидаемый результат:**
- Полный список PTP функций с адресами
- Алгоритм выбора master
- Специфичные расширения PTP Squared
- Обработка PTP сообщений

**Скрипт:** `analyze_ptp_functions.py`

---

### 2.2 NTP модуль

**Цель:** Анализ ключевых NTP функций (из 396)

**Шаги:**

1. **Найти ключевые функции**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i ntp | grep -E "(client|server|sync|adjust|filter|select)" | head -30
```

2. **Проанализировать алгоритмы синхронизации**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i ntp | grep -E "(clock|offset|jitter|dispersion)" | head -20
```

3. **Найти обработку NTP пакетов**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -E "(NTP|ntp)" | head -20
```

4. **Проанализировать фильтры**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i ntp | grep -i filter | head -10
```

**Ожидаемый результат:**
- Ключевые NTP функции с адресами
- Алгоритмы синхронизации
- Фильтры и селекция источников
- Обработка NTP пакетов

**Скрипт:** `analyze_ntp_functions.py`

---

## 🎯 Этап 3: Структуры данных (90% → 95%)

### 3.1 Структуры servo

**Цель:** Полные определения структур

**Шаги:**

1. **Найти структуру AlgoPID**
```bash
# Поиск через смещения в ассемблере
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -E "ldr.*\[.*#[0-9]+\]" | grep -A 1 "41c8680" | head -50
```

2. **Найти структуру Controller**
```bash
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -E "ldr.*\[.*#[0-9]+\]" | grep -A 1 "41e7ff0" | head -50
```

3. **Найти структуру TimeSource**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i "TimeSource"
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 100 "TimeSource" | head -100
```

4. **Анализ смещений**
```bash
# Извлечь все уникальные смещения
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -oE "ldr.*\[.*#([0-9]+|0x[0-9a-f]+)\]" | sort -u | head -100
```

**Ожидаемый результат:**
- Полные определения структур
- Размеры структур
- Выравнивания полей
- Типы полей

**Скрипт:** `extract_data_structures.py`

---

### 3.2 Структуры PTP/NTP

**Аналогично servo**

---

## 🎯 Этап 4: CLI и конфигурация (95% → 98%)

### 4.1 CLI обработчики

**Цель:** Полный анализ CLI интерфейса

**Шаги:**

1. **Найти обработчики команд**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -E "(set|show|command|cli|cmd)" | head -50
```

2. **Найти строки команд**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -E "^(set|show)" | head -30
```

3. **Проанализировать автодополнение**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i "complete\|tab\|suggest" | head -20
```

4. **Найти структуру команд**
```bash
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 50 "set\|show" | head -100
```

**Ожидаемый результат:**
- Все обработчики команд
- Структура дерева команд
- Автодополнение
- Валидация параметров

**Скрипт:** `analyze_cli_handlers.py`

---

### 4.2 Парсинг конфигурации

**Цель:** Полный анализ парсинга YAML

**Шаги:**

1. **Найти YAML парсер**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i yaml | head -20
```

2. **Найти строки конфигурации**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -E "(clock_sync|ptp|ntp|gnss|servo)" | head -30
```

3. **Проанализировать валидацию**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i "valid\|check\|verify" | head -20
```

4. **Найти дефолтные значения**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -E "default|Default" | head -20
```

**Ожидаемый результат:**
- YAML парсер
- Все параметры конфигурации
- Валидация
- Дефолтные значения

**Скрипт:** `analyze_config_parsing.py`

---

## 🎯 Этап 5: Финальная проверка (98% → 100%)

### 5.1 Мониторинг и метрики

**Цель:** Анализ экспорта метрик

**Шаги:**

1. **Найти Elasticsearch экспорт**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i "elastic\|elasticsearch" | head -20
```

2. **Найти метрики**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i "metric\|gauge\|counter" | head -20
```

3. **Найти Grafana интеграцию**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -i grafana | head -10
```

**Скрипт:** `analyze_monitoring.py`

---

### 5.2 TAAS (Time as a Service)

**Цель:** Анализ TAAS протокола

**Шаги:**

1. **Найти TAAS функции**
```bash
nm -D /usr/share/shiwatime/bin/shiwatime | grep -i taas | head -20
```

2. **Найти протокол**
```bash
strings /usr/share/shiwatime/bin/shiwatime | grep -i taas | head -10
```

**Скрипт:** `analyze_taas.py`

---

### 5.3 Тестирование

**Цель:** Создание тестов

**Шаги:**

1. Создать unit тесты для всех модулей
2. Проверить совместимость с оригиналом
3. Валидировать алгоритмы

---

## 📊 Приоритеты

### Высокий приоритет (для 90%)

1. ✅ **PID коэффициенты** - критично для работы
2. ✅ **PI коэффициенты** - критично для работы
3. ✅ **LinReg параметры** - важно для точности

### Средний приоритет (для 95%)

4. ⚠️ **PTP детали** - можно использовать стандартную библиотеку
5. ⚠️ **NTP детали** - можно использовать стандартную библиотеку
6. ⚠️ **Структуры данных** - можно вывести из использования

### Низкий приоритет (для 100%)

7. ❓ **CLI обработчики** - можно создать свой интерфейс
8. ❓ **Мониторинг** - можно добавить позже
9. ❓ **TAAS** - опциональная функция

---

## 🚀 Быстрый старт

### Для достижения 90% (коэффициенты)

```bash
# 1. Создать скрипты
python3 extract_pid_coefficients.py
python3 extract_pi_coefficients.py
python3 extract_linreg_parameters.py

# 2. Запустить на устройстве
scp extract_*.py shiwa@grandmini:~/
ssh shiwa@grandmini
python3 extract_pid_coefficients.py
python3 extract_pi_coefficients.py
python3 extract_linreg_parameters.py

# 3. Скопировать результаты
scp shiwa@grandmini:~/pid_coefficients.txt .
scp shiwa@grandmini:~/pi_coefficients.txt .
scp shiwa@grandmini:~/linreg_parameters.txt .
```

---

## 📝 Чеклист

### Этап 1: Коэффициенты (85% → 90%)
- [ ] PID коэффициенты найдены
- [ ] PI коэффициенты найдены
- [ ] LinReg параметры найдены
- [ ] Скрипты созданы и протестированы

### Этап 2: PTP/NTP (60% → 80%)
- [ ] Все PTP функции найдены
- [ ] Master election алгоритм понят
- [ ] Ключевые NTP функции найдены
- [ ] Алгоритмы синхронизации понятны

### Этап 3: Структуры (90% → 95%)
- [ ] Все структуры servo определены
- [ ] Структуры PTP/NTP определены
- [ ] Размеры и выравнивания известны

### Этап 4: CLI/Config (95% → 98%)
- [ ] CLI обработчики найдены
- [ ] Парсинг конфигурации понят
- [ ] Валидация параметров известна

### Этап 5: Финальная проверка (98% → 100%)
- [ ] Мониторинг проанализирован
- [ ] TAAS проанализирован
- [ ] Тесты созданы

---

**Текущий статус: 85% → Цель: 100%**

*Следуйте плану по этапам для достижения 100% готовности!*
