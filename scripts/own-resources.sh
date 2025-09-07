#!/bin/bash

# This script sets ownership of the resources directory to the current user in case
# docker created files as a different user

sudo chown -R $USER:$USER resources/