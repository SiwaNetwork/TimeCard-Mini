#!/bin/bash
if [ "$PS1" ]; then

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BOLD='\033[1m'
NC='\033[0m'

# Clear screen
clear

# Animated logo display
echo -e "${CYAN}${BOLD}"
sleep 0.05
echo "╔═══════════════════════════════════════════════════════════╗"
sleep 0.05
echo "║                                                           ║"
sleep 0.05
echo "║        ███████╗██╗  ██╗██╗██╗    ██╗ █████╗              ║"
sleep 0.05
echo "║        ██╔════╝██║  ██║██║██║    ██║██╔══██╗             ║"
sleep 0.05
echo "║        ███████╗███████║██║██║ █╗ ██║███████║             ║"
sleep 0.05
echo "║        ╚════██║██╔══██║██║██║███╗██║██╔══██║             ║"
sleep 0.05
echo "║        ███████║██║  ██║██║╚███╔███╔╝██║  ██║             ║"
sleep 0.05
echo "║        ╚══════╝╚═╝  ╚═╝╚═╝ ╚══╝╚══╝ ╚═╝  ╚═╝             ║"
sleep 0.05
echo "║        ████████╗██╗███╗   ███╗███████╗                   ║"
sleep 0.05
echo "║        ╚══██╔══╝██║████╗ ████║██╔════╝                   ║"
sleep 0.05
echo "║           ██║   ██║██╔████╔██║█████╗                     ║"
sleep 0.05
echo "║           ██║   ██║██║╚██╔╝██║██╔══╝                     ║"
sleep 0.05
echo "║           ██║   ██║██║ ╚═╝ ██║███████╗                   ║"
sleep 0.05
echo "║           ╚═╝   ╚═╝╚═╝     ╚═╝╚══════╝                   ║"
sleep 0.05
echo "║                                                           ║"
sleep 0.05
echo "║              QUANTUM MINI-PCI TIMECARD                    ║"
sleep 0.05
echo "║                                                           ║"
sleep 0.05
echo "║         Precision Time Protocol Grandmaster               ║"
sleep 0.05
echo "║            GPS/GLONASS Synchronized Clock                 ║"
sleep 0.05
echo "║                                                           ║"
sleep 0.05
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"
sleep 0.1

# System status
echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
echo -e "${WHITE}  System:${NC}     $(uname -srm)"
echo -e "${WHITE}  Uptime:${NC}     $(uptime -p | sed 's/up //')"
echo -e "${WHITE}  IP:${NC}        $(hostname -I | awk '{print $1}')"

# Check shiwatime status
if systemctl is-active --quiet shiwatime; then
    echo -e "${WHITE}  Shiwa Time:${NC} ${GREEN}● Running${NC}"
else
    echo -e "${WHITE}  Shiwa Time:${NC} ${RED}● Stopped${NC}"
fi

# Check GNSS
if [ -c /dev/ttyS0 ]; then
    echo -e "${WHITE}  GNSS:${NC}       ${GREEN}● Connected${NC}"
else
    echo -e "${WHITE}  GNSS:${NC}       ${RED}● Not found${NC}"
fi

echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${CYAN}Quick commands:${NC}"
echo -e "  ${GREEN}status${NC}   - Check Shiwa Time status"
echo -e "  ${GREEN}logs${NC}     - View logs"
echo -e "  ${GREEN}config${NC}   - Edit configuration"
echo -e "  ${GREEN}restart${NC}  - Restart service"
echo ""
fi
