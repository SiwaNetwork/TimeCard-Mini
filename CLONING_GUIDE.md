# 🔄 Руководство по клонированию Quantum Mini-PCI Timecard

## 📋 Содержание

1. [Поиск устройства в сети](#поиск-устройства-в-сети)
2. [Создание образа системы](#создание-образа-системы)
3. [Восстановление из образа](#восстановление-из-образа)
4. [Настройка после клонирования](#настройка-после-клонирования)
5. [Автоматические скрипты](#автоматические-скрипты)

---

## 🔍 Поиск устройства в сети

### Автоматический поиск устройства

Создайте скрипт для автоматического поиска:

```bash
#!/bin/bash
# find-quantum-device.sh

echo "🔍 Поиск Quantum Mini-PCI Timecard в сети..."

# Определить локальную сеть
LOCAL_IP=$(ip route get 1.1.1.1 | awk '{print $7}' | head -1)
NETWORK=$(echo $LOCAL_IP | cut -d'.' -f1-3).0/24

echo "📡 Сканирование сети: $NETWORK"

# Поиск устройств с SSH
echo "🔎 Поиск устройств с SSH..."
nmap -p 22 --open $NETWORK | grep -E "Nmap scan report|22/tcp open" | \
while read line; do
    if [[ $line == *"Nmap scan report"* ]]; then
        IP=$(echo $line | awk '{print $5}')
    elif [[ $line == *"22/tcp open"* ]]; then
        echo "🖥️  Найдено SSH устройство: $IP"
        
        # Попытка подключения с проверкой hostname
        if timeout 5 ssh -o ConnectTimeout=3 -o StrictHostKeyChecking=no shiwa@$IP "hostname" 2>/dev/null | grep -q "grandmini\|quantum"; then
            echo "✅ Найден Quantum Mini-PCI: $IP"
            echo "🔗 Подключение: ssh shiwa@$IP"
            echo "🔑 Пароль: 278934"
        fi
    fi
done
```

### Ручной поиск

```bash
# Сканирование конкретной сети
nmap -sn 192.168.1.0/24
nmap -sn 192.168.16.0/24
nmap -sn 10.0.0.0/24

# Поиск по имени хоста
nslookup grandmini
ping grandmini.local

# Поиск в ARP таблице
arp -a | grep -i "grandmini\|quantum"
```

---

## 💾 Создание образа системы

### Метод 1: Полный образ SD-карты

```bash
#!/bin/bash
# create-full-backup.sh

# Определить устройство SD-карты
echo "🔍 Поиск SD-карты..."
lsblk | grep -E "sd[a-z].*disk"

read -p "Введите устройство SD-карты (например, /dev/sdb): " DEVICE

if [ ! -b "$DEVICE" ]; then
    echo "❌ Устройство $DEVICE не найдено!"
    exit 1
fi

# Создать имя файла с датой
BACKUP_NAME="quantum-mini-pci-$(date +%Y%m%d-%H%M%S)"

echo "📦 Создание образа: $BACKUP_NAME.img"
echo "⚠️  Это может занять 10-30 минут..."

# Создать образ
sudo dd if=$DEVICE of=${BACKUP_NAME}.img bs=4M status=progress

# Сжать образ
echo "🗜️  Сжатие образа..."
gzip ${BACKUP_NAME}.img

# Создать контрольную сумму
echo "🔐 Создание контрольной суммы..."
sha256sum ${BACKUP_NAME}.img.gz > ${BACKUP_NAME}.img.gz.sha256

echo "✅ Готово!"
echo "📁 Файлы:"
echo "   - ${BACKUP_NAME}.img.gz"
echo "   - ${BACKUP_NAME}.img.gz.sha256"
echo "📊 Размер: $(du -h ${BACKUP_NAME}.img.gz | cut -f1)"
```

### Метод 2: Архив конфигурации

```bash
#!/bin/bash
# create-config-backup.sh

# Найти устройство в сети
echo "🔍 Поиск Quantum Mini-PCI..."
DEVICE_IP=$(nmap -p 22 --open 192.168.16.0/24 | grep -B1 "22/tcp open" | grep "Nmap scan report" | awk '{print $5}' | head -1)

if [ -z "$DEVICE_IP" ]; then
    echo "❌ Quantum Mini-PCI не найден в сети!"
    exit 1
fi

echo "✅ Найден: $DEVICE_IP"

# Создать архив конфигурации
BACKUP_NAME="quantum-config-$(date +%Y%m%d-%H%M%S)"

echo "📦 Создание архива конфигурации..."

ssh shiwa@$DEVICE_IP "sudo tar -czf /tmp/${BACKUP_NAME}.tar.gz \
  /etc/shiwatime/ \
  /etc/systemd/system/shiwatime.service \
  /etc/profile.d/quantum-boot.sh \
  /home/shiwa/.bashrc \
  /boot/config.txt \
  /etc/ssh/sshd_config \
  /etc/hostname \
  /etc/hosts"

# Скачать архив
echo "⬇️  Скачивание архива..."
scp shiwa@$DEVICE_IP:/tmp/${BACKUP_NAME}.tar.gz ./

# Создать контрольную сумму
sha256sum ${BACKUP_NAME}.tar.gz > ${BACKUP_NAME}.tar.gz.sha256

echo "✅ Готово!"
echo "📁 Файлы:"
echo "   - ${BACKUP_NAME}.tar.gz"
echo "   - ${BACKUP_NAME}.tar.gz.sha256"
```

---

## 🔄 Восстановление из образа

### Восстановление полного образа

```bash
#!/bin/bash
# restore-full-backup.sh

# Найти образ
echo "🔍 Поиск образов..."
ls -la *.img.gz 2>/dev/null || echo "❌ Образы не найдены!"

read -p "Введите имя образа (без .gz): " IMAGE_NAME

if [ ! -f "${IMAGE_NAME}.gz" ]; then
    echo "❌ Файл ${IMAGE_NAME}.gz не найден!"
    exit 1
fi

# Проверить контрольную сумму
if [ -f "${IMAGE_NAME}.gz.sha256" ]; then
    echo "🔐 Проверка контрольной суммы..."
    sha256sum -c ${IMAGE_NAME}.gz.sha256
    if [ $? -ne 0 ]; then
        echo "❌ Ошибка контрольной суммы!"
        exit 1
    fi
fi

# Определить устройство SD-карты
echo "🔍 Поиск SD-карты..."
lsblk | grep -E "sd[a-z].*disk"

read -p "Введите устройство SD-карты (например, /dev/sdb): " DEVICE

if [ ! -b "$DEVICE" ]; then
    echo "❌ Устройство $DEVICE не найдено!"
    exit 1
fi

echo "⚠️  ВНИМАНИЕ! Все данные на $DEVICE будут удалены!"
read -p "Продолжить? (y/N): " CONFIRM

if [ "$CONFIRM" != "y" ]; then
    echo "❌ Отменено"
    exit 1
fi

# Распаковать образ
echo "📦 Распаковка образа..."
gunzip ${IMAGE_NAME}.gz

# Записать образ
echo "💾 Запись образа на $DEVICE..."
echo "⏱️  Это может занять 10-30 минут..."
sudo dd if=${IMAGE_NAME} of=$DEVICE bs=4M status=progress

# Синхронизировать
sync

echo "✅ Готово! SD-карта готова к использованию."
```

### Восстановление конфигурации

```bash
#!/bin/bash
# restore-config-backup.sh

# Найти архив
echo "🔍 Поиск архивов конфигурации..."
ls -la *config*.tar.gz 2>/dev/null || echo "❌ Архивы не найдены!"

read -p "Введите имя архива: " ARCHIVE_NAME

if [ ! -f "$ARCHIVE_NAME" ]; then
    echo "❌ Файл $ARCHIVE_NAME не найден!"
    exit 1
fi

# Найти устройство в сети
echo "🔍 Поиск Quantum Mini-PCI..."
DEVICE_IP=$(nmap -p 22 --open 192.168.16.0/24 | grep -B1 "22/tcp open" | grep "Nmap scan report" | awk '{print $5}' | head -1)

if [ -z "$DEVICE_IP" ]; then
    echo "❌ Quantum Mini-PCI не найден в сети!"
    exit 1
fi

echo "✅ Найден: $DEVICE_IP"

# Загрузить архив
echo "⬆️  Загрузка архива..."
scp $ARCHIVE_NAME shiwa@$DEVICE_IP:/tmp/

# Восстановить конфигурацию
echo "🔄 Восстановление конфигурации..."
ssh shiwa@$DEVICE_IP "sudo tar -xzf /tmp/$ARCHIVE_NAME -C /"

# Перезагрузить systemd
echo "🔄 Перезагрузка systemd..."
ssh shiwa@$DEVICE_IP "sudo systemctl daemon-reload"

# Включить сервисы
echo "▶️  Включение сервисов..."
ssh shiwa@$DEVICE_IP "sudo systemctl enable shiwatime"
ssh shiwa@$DEVICE_IP "sudo systemctl start shiwatime"

echo "✅ Конфигурация восстановлена!"
```

---

## ⚙️ Настройка после клонирования

### Автоматическая настройка

```bash
#!/bin/bash
# post-clone-setup.sh

# Найти устройство
DEVICE_IP=$(nmap -p 22 --open 192.168.16.0/24 | grep -B1 "22/tcp open" | grep "Nmap scan report" | awk '{print $5}' | head -1)

if [ -z "$DEVICE_IP" ]; then
    echo "❌ Quantum Mini-PCI не найден!"
    exit 1
fi

echo "✅ Найден: $DEVICE_IP"

# Получить новое имя устройства
read -p "Введите новое имя устройства: " NEW_HOSTNAME

# Настроить устройство
ssh shiwa@$DEVICE_IP << EOF
# Изменить hostname
sudo hostnamectl set-hostname $NEW_HOSTNAME

# Обновить /etc/hosts
sudo sed -i "s/grandmini/$NEW_HOSTNAME/g" /etc/hosts

# Удалить старые SSH ключи
sudo rm -f /etc/ssh/ssh_host_*

# Сгенерировать новые SSH ключи
sudo dpkg-reconfigure -f noninteractive openssh-server

# Сменить пароль пользователя
echo "Смена пароля пользователя shiwa:"
passwd

# Сменить пароль root
echo "Смена пароля root:"
sudo passwd root

# Проверить сервисы
echo "Проверка сервисов:"
sudo systemctl status shiwatime --no-pager -l

echo "✅ Настройка завершена!"
echo "🔄 Рекомендуется перезагрузка: sudo reboot"
EOF
```

---

## 🤖 Автоматические скрипты

### Установка всех скриптов

```bash
#!/bin/bash
# install-cloning-scripts.sh

echo "📦 Установка скриптов клонирования..."

# Создать директорию
mkdir -p ~/quantum-cloning-tools
cd ~/quantum-cloning-tools

# Скачать скрипты (если они в репозитории)
# wget https://raw.githubusercontent.com/your-repo/quantum-mini-pci/main/scripts/find-quantum-device.sh
# wget https://raw.githubusercontent.com/your-repo/quantum-mini-pci/main/scripts/create-full-backup.sh
# и т.д.

# Сделать исполняемыми
chmod +x *.sh

echo "✅ Скрипты установлены в ~/quantum-cloning-tools/"
echo "📖 Использование:"
echo "   ./find-quantum-device.sh    - Найти устройство"
echo "   ./create-full-backup.sh     - Создать полный образ"
echo "   ./create-config-backup.sh   - Создать архив конфигурации"
echo "   ./restore-full-backup.sh    - Восстановить полный образ"
echo "   ./restore-config-backup.sh  - Восстановить конфигурацию"
echo "   ./post-clone-setup.sh       - Настроить после клонирования"
```

### Быстрые команды

Добавьте в ~/.bashrc:

```bash
# Quantum Mini-PCI клонирование
alias find-quantum='~/quantum-cloning-tools/find-quantum-device.sh'
alias backup-quantum='~/quantum-cloning-tools/create-full-backup.sh'
alias restore-quantum='~/quantum-cloning-tools/restore-full-backup.sh'
alias setup-quantum='~/quantum-cloning-tools/post-clone-setup.sh'
```

---

## 📋 Чек-лист клонирования

### Перед клонированием

- [ ] Устройство настроено и работает корректно
- [ ] Все сервисы запущены (`status` показывает "Running")
- [ ] Конфигурация протестирована
- [ ] Создан бэкап важных данных

### После клонирования

- [ ] Изменен hostname устройства
- [ ] Обновлены SSH ключи
- [ ] Изменены пароли по умолчанию
- [ ] Проверена работа всех сервисов
- [ ] Настроены сетевые параметры
- [ ] Протестирована синхронизация времени

### Проверка работоспособности

```bash
# Подключиться к клонированному устройству
ssh shiwa@NEW_IP

# Проверить статус
status

# Проверить логи
logs

# Проверить синхронизацию
chronyc sources -v
```

---

## 🔗 Связанные документы

- [README.md](README.md) - Основная документация
- [SHIWATIME_GUIDE.md](SHIWATIME_GUIDE.md) - Руководство по Shiwa Time
- [MONITORING.md](MONITORING.md) - Мониторинг системы

---

**Дата создания:** $(date +%Y-%m-%d)  
**Версия:** 1.0  
**Статус:** Готово к использованию
