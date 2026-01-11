#!/bin/bash
# Скрипт для автоматического поиска Quantum Mini-PCI Timecard в сети

echo "🔍 Поиск Quantum Mini-PCI Timecard в сети..."
echo "=============================================="

# Определить локальную сеть
LOCAL_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '{print $7}' | head -1)

if [ -z "$LOCAL_IP" ]; then
    echo "❌ Не удалось определить локальную сеть"
    echo "💡 Попробуйте указать сеть вручную:"
    echo "   ./find-quantum-device.sh 192.168.1.0/24"
    exit 1
fi

# Определить сеть для сканирования
if [ -n "$1" ]; then
    NETWORK="$1"
    echo "📡 Сканирование указанной сети: $NETWORK"
else
    NETWORK=$(echo $LOCAL_IP | cut -d'.' -f1-3).0/24
    echo "📡 Автоматическое определение сети: $NETWORK"
fi

echo ""

# Проверить наличие nmap
if ! command -v nmap &> /dev/null; then
    echo "❌ nmap не установлен!"
    echo "💡 Установите nmap:"
    echo "   Ubuntu/Debian: sudo apt install nmap"
    echo "   CentOS/RHEL: sudo yum install nmap"
    echo "   macOS: brew install nmap"
    exit 1
fi

# Поиск устройств с SSH
echo "🔎 Поиск устройств с SSH..."
echo "⏱️  Это может занять 30-60 секунд..."
echo ""

FOUND_DEVICES=()
CURRENT_IP=""

# Сканирование с обработкой вывода
nmap -p 22 --open $NETWORK | while IFS= read -r line; do
    if [[ $line == *"Nmap scan report"* ]]; then
        CURRENT_IP=$(echo $line | awk '{print $5}')
        HOSTNAME=$(echo $line | awk '{print $6}' | tr -d '()')
        echo "🖥️  Найдено SSH устройство: $CURRENT_IP ($HOSTNAME)"
        
        # Попытка подключения с проверкой hostname
        if timeout 5 ssh -o ConnectTimeout=3 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null shiwa@$CURRENT_IP "hostname" 2>/dev/null | grep -q "grandmini\|quantum"; then
            echo "✅ 🎯 НАЙДЕН QUANTUM MINI-PCI: $CURRENT_IP"
            echo "   🔗 Подключение: ssh shiwa@$CURRENT_IP"
            echo "   🔑 Пароль: 278934"
            echo "   🏷️  Hostname: $HOSTNAME"
            echo ""
            
            # Проверить статус сервисов
            echo "📊 Проверка статуса сервисов:"
            if timeout 10 ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null shiwa@$CURRENT_IP "sudo systemctl is-active shiwatime" 2>/dev/null | grep -q "active"; then
                echo "   ✅ Shiwa Time: Running"
            else
                echo "   ❌ Shiwa Time: Stopped"
            fi
            
            if timeout 10 ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null shiwa@$CURRENT_IP "test -c /dev/ttyS0" 2>/dev/null; then
                echo "   ✅ GNSS: Connected"
            else
                echo "   ❌ GNSS: Not found"
            fi
            
            echo ""
            echo "🚀 Быстрое подключение:"
            echo "   ssh shiwa@$CURRENT_IP"
            echo ""
            
        elif timeout 5 ssh -o ConnectTimeout=3 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null shiwa@$CURRENT_IP "echo 'test'" 2>/dev/null >/dev/null; then
            echo "   ⚠️  SSH доступен, но не Quantum Mini-PCI"
        fi
    fi
done

echo "=============================================="
echo "🔍 Поиск завершен!"

# Если ничего не найдено, предложить альтернативы
if [ ${#FOUND_DEVICES[@]} -eq 0 ]; then
    echo ""
    echo "💡 Альтернативные способы поиска:"
    echo "   1. Проверьте ARP таблицу: arp -a"
    echo "   2. Попробуйте другую сеть: ./find-quantum-device.sh 192.168.1.0/24"
    echo "   3. Поиск по hostname: ping grandmini.local"
    echo "   4. Проверьте роутер: обычно устройства отображаются в веб-интерфейсе"
fi

echo ""
echo "📖 Дополнительная информация:"
echo "   - Документация: README.md"
echo "   - Клонирование: CLONING_GUIDE.md"
echo "   - Мониторинг: MONITORING.md"
