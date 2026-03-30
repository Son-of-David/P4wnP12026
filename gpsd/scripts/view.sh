#!/bin/bash
systemctl restart gpsd.socket
systemctl restart gpsd
systemctl restart GPSD
gpsmon

