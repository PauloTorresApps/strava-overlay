#!/bin/sh
mkdir -p scripts
cat > scripts/postinstall.sh << 'EOF'
#!/bin/sh
echo "Atualizando cache de ícones e aplicações..."
gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor 2>/dev/null || true
update-desktop-database -q 2>/dev/null || true
echo "Instalação concluída!"
EOF
chmod +x scripts/postinstall.sh# scripts/postinstall.sh
echo "Atualizando cache de ícones e aplicações..."
gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor
update-desktop-database -q
echo "Concluído."