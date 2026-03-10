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

FROM php:8.2-apache

# Install mysqli
RUN docker-php-ext-install mysqli

# Enable Apache modules
RUN a2enmod rewrite ssl

# Apache config
COPY config/000-default.conf /etc/apache2/sites-available/000-default.conf
COPY config/default-ssl.conf /etc/apache2/sites-available/default-ssl.conf

RUN a2ensite 000-default.conf \
    && a2ensite default-ssl.conf

# App code
COPY . /var/www/html/

# Copy HTTPS certs
COPY ssl/server.crt /etc/apache2/ssl/server.crt
COPY ssl/server.key /etc/apache2/ssl/server.key

# Fix permissions
RUN chown -R www-data:www-data /var/www/html \
    && chmod -R 755 /var/www/html \
    && chmod 644 /var/www/html/certs/ca.pem /var/www/html/certs/client-cert.pem \
    && chmod 600 /var/www/html/certs/client-key.pem \
    && chmod 644 /etc/apache2/ssl/server.crt \
    && chmod 600 /etc/apache2/ssl/server.key

EXPOSE 80 443

CMD ["apache2-foreground"]
