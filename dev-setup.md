
# Generic Logger -- Development Setup Guide

  

## Overview

  

This document describes how to configure and provision the Generic

Logger application in a development environment. It explains each

DocType, every field within those DocTypes, and the correct provisioning

order.

  

------------------------------------------------------------------------

  

# 1. QuickWit Server

  

Represents a Quickwit instance and its associated authorization proxy

running on a Virtual Machine.

  

This should be created and provisioned first.

  

## Fields

  

### Endpoint URL

  

S3-compatible storage endpoint used by Quickwit.

  

Examples: - https://s3.amazonaws.com - http://10.0.0.5:9000

  

This must be reachable from the Quickwit VM.

  

### S3 Bucket

  

Name of the bucket where Quickwit stores index data and metadata.

  

Example: - quickwit-logs

  

### S3 Access Token

  

Access key for the S3-compatible storage. Stored securely.

  

### S3 Secret Key

  

Secret key for the S3-compatible storage. Stored securely.

  

### Region

  

S3 region.

  

Example: - us-east-1

  

### Virtual Machine

  

The VM where Quickwit and the authorization proxy will be deployed.

  

Expected services: - Quickwit on port 7280 - Proxy on port 8080

  

### Grafana VM

  

The VM where Grafana is running.

  

Used to configure internal networking and JWT validation against

Grafana's signing keys.

  

## Provision Action

  

After saving the document, click "Provision".

  

This will: - Deploy Quickwit - Deploy the authorization proxy -

Configure environment variables - Configure system supervisor - Expose

proxy on port 8080

  

------------------------------------------------------------------------

  

# 2. QuickWit Index

  

Represents a Quickwit index created on a specific QuickWit Server.

  

Create this after provisioning QuickWit Server.

  

## Fields

  

### Schema (JSON)

  

Complete Quickwit index schema definition.

  

Example:

  

{ "version": "0.7", "index_id": "docker_logs", "doc_mapping": {

"timestamp_field": "time", "partition_key": "tenant_id" } }

  

Notes: - index_id becomes the document name. - timestamp_field must

match the log timestamp field. - partition_key should match the tenant

isolation field.

  

### QuickWit Server

  

Link to the QuickWit Server where this index will be created.

  

### Created (Hidden)

  

Automatically managed flag: - 1 after successful creation - 0 after

deletion

  

Do not modify manually.

  

## Actions

  

### Create

  

Creates the index using the Quickwit API.

  

### Delete

  

Deletes the index from Quickwit and resets the Created flag.

  

------------------------------------------------------------------------

  

# 3. Grafana Server

  

Represents a Grafana instance running on a Virtual Machine.

  

Create this after QuickWit Index has been successfully created.

  

## Fields

  

### Admin User

  

Grafana administrator username.

  

Example: - admin

  

### Admin Password

  

Grafana administrator password.

  

Used during provisioning and stored securely.

  

### Virtual Machine

  

The VM where Grafana will run.

  

Grafana will be accessible at: http://`<vm-ip>`{=html}:3000

  

### QuickWit Index

  

The index this Grafana instance will query.

  

Used to configure the datasource and connect to the proxy.

  

### OAuth (Read Only)

  

Automatically created during save if not already present.

  

An OAuth Client is created with: - Redirect URI:

http://`<grafana-ip>`{=html}:3000/login/generic_oauth - Scopes: openid

profile user:email

  

No manual configuration required.

  

## Provision Action

  

After saving the document, click "Provision".

  

This will: - Deploy Grafana - Configure Generic OAuth against Frappe -

Configure datasource pointing to proxy (port 8080) - Enable OAuth

forwarding - Configure root URL and domain settings

  

------------------------------------------------------------------------

  

# 4. Log User

  

Maps a Frappe User to a tenant identifier.

  

Used by the proxy to enforce multi-tenant log isolation.

  

## Fields

  

### User

  

Link to an existing Frappe User (email).

  

Must be unique.

  

### Tenant ID

  

Tenant identifier used in log documents.

  

Example: - tenant_1

  

This value must match the tenant_id field stored inside indexed logs.

  

------------------------------------------------------------------------

  

# Authorization Flow (Reference)

  

1. User logs into Grafana using Frappe OAuth.

2. Grafana forwards the ID token (JWT) in request headers.

3. Proxy validates JWT using Grafana JWKS.

4. Proxy extracts email.

5. Proxy calls Frappe API to fetch Log User document.

6. Proxy injects tenant-based filters into queries.

7. Quickwit returns only authorized results.

  

------------------------------------------------------------------------

  

# Provisioning Order

  

Follow this sequence strictly:

  

1. Create and Provision QuickWit Server

2. Create and Create QuickWit Index

3. Create and Provision Grafana Server

4. Create Log User entries

  

------------------------------------------------------------------------

  

# Development Checklist

  

Before testing:

  

- Quickwit reachable on port 7280

- Proxy reachable on port 8080

- Grafana reachable on port 3000

- QuickWit Index successfully created

- Log User entry exists

- tenant_id in logs matches Log User configuration

  

------------------------------------------------------------------------

  

## Overview

  

This document describes how to configure and provision the Generic

Logger application in a development environment. It explains each

DocType, every field within those DocTypes, and the correct provisioning

order.


  

------------------------------------------------------------------------
