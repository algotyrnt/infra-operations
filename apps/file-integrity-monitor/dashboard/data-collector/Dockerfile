#   Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).

#  WSO2 LLC. licenses this file to you under the Apache License,
#  Version 2.0 (the "License"); you may not use this file except
#  in compliance with the License.
#  You may obtain a copy of the License at

#  http://www.apache.org/licenses/LICENSE-2.0

#  Unless required by applicable law or agreed to in writing,
#  software distributed under the License is distributed on an
#  "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
#  KIND, either express or implied.  See the License for the
#  specific language governing permissions and limitations
#  under the License. 


# Use official slim Python image
FROM python:3.11-slim

# Ensure Python outputs logs immediately
ENV PYTHONUNBUFFERED=1

# Set working directory
WORKDIR /app

# Copy application code
COPY app/ /app/

# Copy TLS certs into container
COPY certs/ /app/certs/

# Install Python dependencies and create runtime folder
RUN pip install --no-cache-dir boto3 mysql-connector-python \
    && mkdir -p /app/centralised_bak \
    && chmod 600 /app/certs/client-key.pem \
    && chmod 644 /app/certs/ca.pem /app/certs/client-cert.pem

# Run the main Python script
CMD ["python", "S3toDB.py"]
