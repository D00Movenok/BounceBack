#!/bin/bash

ipRegex="((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(/[0-9]?[0-9])?)";
curl https://gist.githubusercontent.com/curi0usJack/971385e8334e189d93a6cb4671238b10/raw | awk '/BURN AV BURN/ {seen = 1} seen {print}' | grep -oE "(^\s$)|(#.*)|${ipRegex}";

echo -e "\n# TOR exit nodes (https://github.com/SecOps-Institute/Tor-IP-Addresses/)";
curl https://raw.githubusercontent.com/SecOps-Institute/Tor-IP-Addresses/master/tor-exit-nodes.lst;

echo -e "\n# SPAMHAUS (https://github.com/SecOps-Institute/SpamhausIPLists/)";
curl https://raw.githubusercontent.com/SecOps-Institute/SpamhausIPLists/master/drop.txt;
curl https://raw.githubusercontent.com/SecOps-Institute/SpamhausIPLists/master/edrop.txt;
curl https://raw.githubusercontent.com/SecOps-Institute/SpamhausIPLists/master/drop_ipv6.txt;

echo -e "\n# CLOUDS and BOTS (https://github.com/lord-alfred/ipranges)"
curl https://raw.githubusercontent.com/lord-alfred/ipranges/main/all/ipv4_merged.txt;
curl https://raw.githubusercontent.com/lord-alfred/ipranges/main/all/ipv6_merged.txt;
 
echo -e "\n# BAD IP list (https://github.com/stamparm/ipsum)"
curl https://raw.githubusercontent.com/stamparm/ipsum/master/levels/3.txt;
