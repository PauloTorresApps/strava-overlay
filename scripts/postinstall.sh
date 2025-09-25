#!/bin/sh
# scripts/postinstall.sh
echo "Atualizando cache de ícones e aplicações..."
gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
update-desktop-database -q
echo "Concluído."