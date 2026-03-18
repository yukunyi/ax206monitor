#!/bin/bash

# MetricsRenderSender - Build Script
# Uses common build functions to reduce code duplication

source ./build_common.sh

echo "MetricsRenderSender - Build Script"
echo "=================================="

# Execute common build steps
common_build

# Show results
show_build_results
