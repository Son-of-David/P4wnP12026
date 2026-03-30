#!/bin/bash

systemctl enable GPSD.service
systemctl start GPSD.service
systemctl restart gpsd.socket
systemctl restart gpsd
