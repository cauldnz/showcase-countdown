#!/bin/sh
# One-shot setup of a freshly flashed vanilla OpenWrt GL-MT3000 as the event
# router: WiFi, NTP server for the LAN, and the showcase service.
# Run ON THE ROUTER as root after `install.sh` has copied the binary, or run
# it first and install afterwards; order does not matter.
#
#   sh router-setup.sh "<ssid>" "<wifi key>" [channel]
#
# Idempotent: re-running just re-applies the same settings.
set -eu

SSID="${1:?usage: router-setup.sh <ssid> <wifi-key> [channel]}"
KEY="${2:?wifi key}"
CHANNEL="${3:-6}"

echo "== wifi: 2.4 GHz '$SSID' on channel $CHANNEL (fixed, for ESP-NOW); 5 GHz same SSID for laptops"
# radio0 is 2.4 GHz and radio1 is 5 GHz on the MT3000 in vanilla OpenWrt.
uci set wireless.radio0.channel="$CHANNEL"
uci set wireless.radio0.htmode='HT20'
uci set wireless.radio0.disabled='0'
uci set wireless.radio0.country='AU'
uci set wireless.default_radio0.ssid="$SSID"
uci set wireless.default_radio0.encryption='psk2'
uci set wireless.default_radio0.key="$KEY"
uci set wireless.default_radio0.network='lan'
uci set wireless.default_radio0.mode='ap'
uci set wireless.radio1.disabled='0'
uci set wireless.radio1.country='AU'
uci set wireless.default_radio1.ssid="$SSID"
uci set wireless.default_radio1.encryption='psk2'
uci set wireless.default_radio1.key="$KEY"
uci set wireless.default_radio1.network='lan'
uci set wireless.default_radio1.mode='ap'
uci commit wireless

echo "== ntp: serve time to the LAN"
uci set system.ntp.enabled='1'
uci set system.ntp.enable_server='1'
uci commit system

echo "== hostname"
uci set system.@system[0].hostname='showcase-router'
uci commit system

echo "== lan stays 192.168.8.1/24 like the GL default, so nothing else changes"
uci set network.lan.ipaddr='192.168.8.1'
uci set network.lan.netmask='255.255.255.0'
uci commit network

echo "== apply"
wifi reload
/etc/init.d/sysntpd restart
/etc/init.d/network reload

echo "== done"
echo "  ssid    : $SSID (2.4 GHz ch $CHANNEL + 5 GHz)"
echo "  lan     : 192.168.8.1"
echo "  ntp     : serving on 192.168.8.1:123"
echo "  next    : sh server/deploy/install.sh from the dev machine, then /etc/init.d/showcase-server status"
