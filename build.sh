#!/bin/bash

# AX206 System Monitor - Build Script
# Uses common build functions to reduce code duplication

source ./build_common.sh

echo "AX206 System Monitor - Build Script"
echo "==================================="

# Execute common build steps
common_build

# Show results
show_build_results