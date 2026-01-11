# Настройка кастомного загрузочного экрана Raspberry Pi
## QUANTUM MINI-PCI Boot Customization

## Содержание

1. [Простой способ: Текстовый логотип при загрузке](#способ-1-текстовый-логотип-ascii-art)
2. [Средний способ: Plymouth Splash Screen](#способ-2-plymouth-splash-screen)
3. [Продвинутый способ: Графический логотип](#способ-3-графический-логотип-kernel-logo)
4. [Настройка MOTD (приветствие при входе)](#настройка-motd-message-of-the-day)
5. [Скрытие системных сообщений](#скрытие-системных-сообщений-при-загрузке)

---

## Способ 1: Текстовый логотип (ASCII Art)

Самый простой способ - показывать текстовый логотип при входе по SSH и на консоли.

### Шаг 1: Создание ASCII логотипа

Подключитесь к устройству:
```bash
ssh shiwa@192.168.16.163
# Пароль: 278934
```

Создайте файл логотипа:
```bash
sudo nano /etc/motd
```

Вставьте логотип (пример):
```
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║        ███████╗██╗  ██╗██╗██╗    ██╗ █████╗              ║
║        ██╔════╝██║  ██║██║██║    ██║██╔══██╗             ║
║        ███████╗███████║██║██║ █╗ ██║███████║             ║
║        ╚════██║██╔══██║██║██║███╗██║██╔══██║             ║
║        ███████║██║  ██║██║╚███╔███╔╝██║  ██║             ║
║        ╚══════╝╚═╝  ╚═╝╚═╝ ╚══╝╚══╝ ╚═╝  ╚═╝             ║
║        ████████╗██╗███╗   ███╗███████╗                   ║
║        ╚══██╔══╝██║████╗ ████║██╔════╝                   ║
║           ██║   ██║██╔████╔██║█████╗                     ║
║           ██║   ██║██║╚██╔╝██║██╔══╝                     ║
║           ██║   ██║██║ ╚═╝ ██║███████╗                   ║
║           ╚═╝   ╚═╝╚═╝     ╚═╝╚══════╝                   ║
║                                                           ║
║              QUANTUM MINI-PCI TIMECARD                    ║
║                                                           ║
║         Precision Time Protocol Grandmaster               ║
║            GPS/GLONASS Synchronized Clock                 ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
```

Или более простой вариант:
```
═══════════════════════════════════════════════════
       QUANTUM MINI-PCI TIMECARD
═══════════════════════════════════════════════════
  
  🕐 Precision Time Server
  📡 PTP Grandmaster
  🛰️  GPS+GLONASS Synchronized
  
  IP: 192.168.16.163
  Status: shiwatime status
  
═══════════════════════════════════════════════════
```

Сохраните: `Ctrl+O`, `Enter`, `Ctrl+X`

### Шаг 2: Динамическое приветствие с информацией

Создайте скрипт для динамической информации:
```bash
sudo nano /etc/profile.d/quantum-welcome.sh
```

Вставьте:
```bash
#!/bin/bash

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

clear

echo -e "${CYAN}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║        ███████╗██╗  ██╗██╗██╗    ██╗ █████╗              ║
║        ██╔════╝██║  ██║██║██║    ██║██╔══██╗             ║
║        ███████╗███████║██║██║ █╗ ██║███████║             ║
║        ╚════██║██╔══██║██║██║███╗██║██╔══██║             ║
║        ███████║██║  ██║██║╚███╔███╔╝██║  ██║             ║
║        ╚══════╝╚═╝  ╚═╝╚═╝ ╚══╝╚══╝ ╚═╝  ╚═╝             ║
║        ████████╗██╗███╗   ███╗███████╗                   ║
║        ╚══██╔══╝██║████╗ ████║██╔════╝                   ║
║           ██║   ██║██╔████╔██║█████╗                     ║
║           ██║   ██║██║╚██╔╝██║██╔══╝                     ║
║           ██║   ██║██║ ╚═╝ ██║███████╗                   ║
║           ╚═╝   ╚═╝╚═╝     ╚═╝╚══════╝                   ║
║                                                           ║
║              QUANTUM MINI-PCI TIMECARD                    ║
║                                                           ║
║         Precision Time Protocol Grandmaster               ║
║            GPS/GLONASS Synchronized Clock                 ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
echo -e "${WHITE}  Система:${NC} $(uname -srm)"
echo -e "${WHITE}  Uptime:${NC}  $(uptime -p | sed 's/up //')"
echo -e "${WHITE}  IP:${NC}     $(hostname -I | awk '{print $1}')"

# Проверка статуса shiwatime
if systemctl is-active --quiet shiwatime; then
    echo -e "${WHITE}  Shiwa Time:${NC} ${GREEN}● Running${NC}"
else
    echo -e "${WHITE}  Shiwa Time:${NC} ${RED}● Stopped${NC}"
fi

# Проверка GNSS
if [ -c /dev/ttyS0 ]; then
    echo -e "${WHITE}  GNSS:${NC}   ${GREEN}● Connected${NC}"
else
    echo -e "${WHITE}  GNSS:${NC}   ${RED}● Not found${NC}"
fi

echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${CYAN}Быстрые команды:${NC}"
echo -e "  ${GREEN}status${NC}  - Статус Shiwa Time"
echo -e "  ${GREEN}logs${NC}    - Просмотр логов"
echo -e "  ${GREEN}config${NC}  - Редактировать конфигурацию"
echo ""
```

Сделайте исполняемым:
```bash
sudo chmod +x /etc/profile.d/quantum-welcome.sh
```

### Шаг 3: Добавьте алиасы команд

```bash
nano ~/.bashrc
```

Добавьте в конец:
```bash
# Quantum Mini-PCI shortcuts
alias status='sudo systemctl status shiwatime'
alias logs='sudo journalctl -u shiwatime -f'
alias config='sudo nano /etc/shiwatime/shiwatime.yml'
alias restart='sudo systemctl restart shiwatime'
```

Применить:
```bash
source ~/.bashrc
```

---

## Способ 2: Plymouth Splash Screen

Plymouth - это система загрузочных экранов для Linux.

### Установка Plymouth

```bash
sudo apt update
sudo apt install plymouth plymouth-themes -y
```

### Создание кастомной темы

**Шаг 1: Создайте директорию темы**
```bash
sudo mkdir -p /usr/share/plymouth/themes/quantum-mini
cd /usr/share/plymouth/themes/quantum-mini
```

**Шаг 2: Создайте файл темы**
```bash
sudo nano quantum-mini.plymouth
```

Содержимое:
```ini
[Plymouth Theme]
Name=Quantum Mini-PCI
Description=Quantum Mini-PCI Timecard Boot Screen
ModuleName=script

[script]
ImageDir=/usr/share/plymouth/themes/quantum-mini
ScriptFile=/usr/share/plymouth/themes/quantum-mini/quantum-mini.script
```

**Шаг 3: Создайте скрипт анимации**
```bash
sudo nano quantum-mini.script
```

Содержимое (простая версия):
```
# Цвета
Window.SetBackgroundTopColor(0.00, 0.00, 0.15);     # Темно-синий
Window.SetBackgroundBottomColor(0.00, 0.00, 0.05);  # Почти черный

# Логотип (если есть изображение)
logo.image = Image("logo.png");
logo.sprite = Sprite(logo.image);
logo.sprite.SetX(Window.GetWidth() / 2 - logo.image.GetWidth() / 2);
logo.sprite.SetY(Window.GetHeight() / 2 - logo.image.GetHeight() / 2 - 50);

# Текст
message_sprite = Sprite();
message_sprite.SetPosition(Window.GetWidth() / 2, Window.GetHeight() / 2 + 100, 10000);

fun message_callback(text) {
    my_image = Image.Text(text, 1, 1, 1);
    message_sprite.SetImage(my_image);
    message_sprite.SetX(Window.GetWidth() / 2 - my_image.GetWidth() / 2);
}

Plymouth.SetMessageFunction(message_callback);

# Прогресс-бар
progress_box.image = Image("progress_box.png");
progress_box.sprite = Sprite(progress_box.image);
progress_box.sprite.SetX(Window.GetWidth() / 2 - progress_box.image.GetWidth() / 2);
progress_box.sprite.SetY(Window.GetHeight() * 0.75);

fun progress_callback(duration, progress) {
    if (progress_bar.image.GetWidth() != Math.Int(progress_box.image.GetWidth() * progress)) {
        progress_bar.image = Image("progress_bar.png").Scale(
            Math.Int(progress_box.image.GetWidth() * progress),
            progress_box.image.GetHeight()
        );
        progress_bar.sprite.SetImage(progress_bar.image);
    }
}

Plymouth.SetBootProgressFunction(progress_callback);
```

**Шаг 4: Создайте простое текстовое изображение (если нет графики)**

Если у вас нет готового логотипа, создайте текстовый:
```bash
# Установите ImageMagick для создания изображений
sudo apt install imagemagick -y

# Создайте текстовый логотип
convert -size 400x100 xc:none -gravity center \
  -fill white -font DejaVu-Sans-Bold -pointsize 32 \
  -annotate +0-20 "QUANTUM" \
  -annotate +0+20 "MINI-PCI" \
  /usr/share/plymouth/themes/quantum-mini/logo.png
```

**Шаг 5: Установите тему**
```bash
sudo plymouth-set-default-theme quantum-mini
sudo update-initramfs -u
```

**Шаг 6: Включите Plymouth в загрузке**
```bash
sudo nano /boot/cmdline.txt
```

Найдите строку и добавьте в конец (если нет):
```
splash quiet plymouth.ignore-serial-consoles
```

Должно получиться что-то вроде:
```
console=serial0,115200 console=tty1 root=PARTUUID=xxxxxxxx-xx rootfstype=ext4 fsck.repair=yes rootwait splash quiet plymouth.ignore-serial-consoles
```

Перезагрузите:
```bash
sudo reboot
```

---

## Способ 3: Графический логотип (Kernel Logo)

Этот способ заменяет логотип Raspberry Pi в самом начале загрузки.

### Подготовка логотипа

**1. Создайте PNG изображение:**
- Размер: 80x80 пикселей (рекомендуется)
- Формат: PNG с прозрачностью
- Цвета: До 224 цветов

**2. Конвертируйте в PPM формат:**
```bash
# На вашем компьютере с установленным imagemagick
convert logo.png -resize 80x80 logo.ppm
```

**3. Конвертируйте в ASCII PPM:**
```bash
convert logo.ppm logo_ascii.ppm
```

### Замена логотипа ядра (требует компиляции ядра)

Это сложный процесс, проще использовать Plymouth или MOTD.

---

## Способ 4: Текстовый логотип при загрузке

Самый простой - добавить вывод логотипа в `/etc/rc.local`:

```bash
sudo nano /etc/rc.local
```

Добавьте перед `exit 0`:
```bash
# Показать логотип на консоли
cat << 'EOF' > /dev/console

╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║     ██████╗ ██╗   ██╗ █████╗ ███╗   ██╗████████╗██╗   ██╗║
║    ██╔═══██╗██║   ██║██╔══██╗████╗  ██║╚══██╔══╝██║   ██║║
║    ██║   ██║██║   ██║███████║██╔██╗ ██║   ██║   ██║   ██║║
║    ██║▄▄ ██║██║   ██║██╔══██║██║╚██╗██║   ██║   ██║   ██║║
║    ╚██████╔╝╚██████╔╝██║  ██║██║ ╚████║   ██║   ╚██████╔╝║
║     ╚══▀▀═╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝    ╚═════╝ ║
║                                                           ║
║              ═══ MINI-PCI TIME CARD ═══                  ║
║                                                           ║
║         Precision Time Protocol Grandmaster              ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

EOF

exit 0
```

---

## Настройка MOTD (Message of the Day)

### Удалить стандартное MOTD

```bash
sudo rm /etc/motd
sudo touch /etc/motd
```

### Отключить динамические скрипты MOTD

```bash
sudo chmod -x /etc/update-motd.d/*
```

### Создать статическое MOTD

```bash
sudo nano /etc/motd
```

Пример содержимого:
```
═══════════════════════════════════════════════════════════════
       
       ██████╗ ██╗   ██╗ █████╗ ███╗   ██╗████████╗██╗   ██╗
      ██╔═══██╗██║   ██║██╔══██╗████╗  ██║╚══██╔══╝██║   ██║
      ██║   ██║██║   ██║███████║██╔██╗ ██║   ██║   ██║   ██║
      ██║▄▄ ██║██║   ██║██╔══██║██║╚██╗██║   ██║   ██║   ██║
      ╚██████╔╝╚██████╔╝██║  ██║██║ ╚████║   ██║   ╚██████╔╝
       ╚══▀▀═╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝    ╚═════╝
                                                           
              ═══ MINI-PCI TIME CARD ═══
       
         🕐 Precision Time Protocol Grandmaster
         📡 GPS/GLONASS Synchronized
         🌐 Network Time Distribution
         
═══════════════════════════════════════════════════════════════

  Commands:
    status  - Check Shiwa Time status
    logs    - View real-time logs
    config  - Edit configuration
    
═══════════════════════════════════════════════════════════════
```

---

## Скрытие системных сообщений при загрузке

Для чистого загрузочного экрана отредактируйте `/boot/cmdline.txt`:

```bash
sudo nano /boot/cmdline.txt
```

Добавьте эти параметры:
```
quiet splash loglevel=3 logo.nologo vt.global_cursor_default=0
```

Параметры:
- `quiet` - минимизировать вывод
- `splash` - показывать splash screen
- `loglevel=3` - только критические сообщения
- `logo.nologo` - скрыть логотип Tux
- `vt.global_cursor_default=0` - скрыть курсор

---

## Генераторы ASCII Art

Онлайн инструменты для создания ASCII логотипов:
- https://patorjk.com/software/taag/
- https://www.ascii-art-generator.org/
- https://www.ascii-art.de/

Выберите шрифт (рекомендуется: ANSI Shadow, Big, Banner3-D)

---

## Быстрый старт (рекомендуемый способ)

Самый простой и быстрый способ для начала:

**1. Создайте приветственный скрипт:**
```bash
sudo nano /etc/profile.d/quantum-boot.sh
```

**2. Вставьте:**
```bash
#!/bin/bash
if [ "$PS1" ]; then
cat << "EOF"

╔═══════════════════════════════════════════════════╗
║         QUANTUM MINI-PCI TIMECARD                ║
║    Precision Time Protocol Grandmaster           ║
╚═══════════════════════════════════════════════════╝

EOF
fi
```

**3. Сделайте исполняемым:**
```bash
sudo chmod +x /etc/profile.d/quantum-boot.sh
```

**4. Выйдите и зайдите снова:**
```bash
exit
ssh shiwa@192.168.16.163
# Пароль: 278934
```

---

## Примеры готовых логотипов

### Вариант 1 (компактный):
```
═══════════════════════════════════════
    QUANTUM MINI-PCI TIMECARD
═══════════════════════════════════════
 PTP Grandmaster | GPS Synchronized
═══════════════════════════════════════
```

### Вариант 2 (с рамкой):
```
┌─────────────────────────────────────┐
│   QUANTUM MINI-PCI TIME CARD       │
│                                     │
│   ⏱️  PTP Grandmaster               │
│   🛰️  GPS+GLONASS Sync              │
│   🌐 Network Time Server            │
└─────────────────────────────────────┘
```

### Вариант 3 (ASCII Art большой):
```
 ██████╗ ██╗   ██╗ █████╗ ███╗   ██╗████████╗██╗   ██╗███╗   ███╗
██╔═══██╗██║   ██║██╔══██╗████╗  ██║╚══██╔══╝██║   ██║████╗ ████║
██║   ██║██║   ██║███████║██╔██╗ ██║   ██║   ██║   ██║██╔████╔██║
██║▄▄ ██║██║   ██║██╔══██║██║╚██╗██║   ██║   ██║   ██║██║╚██╔╝██║
╚██████╔╝╚██████╔╝██║  ██║██║ ╚████║   ██║   ╚██████╔╝██║ ╚═╝ ██║
 ╚══▀▀═╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝    ╚═════╝ ╚═╝     ╚═╝
              MINI-PCI TIME CARD
```
