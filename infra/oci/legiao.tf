# Legiao - relay do jogo online na Oracle Always Free (Sao Paulo).
#
# UM arquivo so, de proposito: o Cloud Shell envia um arquivo por vez.
# Envie tambem o binario `legiao-relay` (gerado a partir de cmd/relay) para a
# MESMA pasta deste arquivo.
#
# Cria: rede publica, firewall (TCP 9000), um bucket privado com o binario do
# relay, um bucket publico com a configuracao que o app le (online.json) e
# UMA VM Always Free (VM.Standard.E2.1.Micro, Ubuntu 22.04). A VM baixa o
# binario do bucket a cada inicio do servico, entao atualizar o relay e:
# enviar o binario novo, `terraform apply` e reiniciar a VM (ver output
# `update_relay`). Nenhum SSH necessario.
#
# Uso no Cloud Shell, na pasta com os dois arquivos:
#   export TF_VAR_tenancy_ocid=$OCI_TENANCY
#   terraform init && terraform plan && terraform apply

terraform {
  required_version = ">= 1.3"
  required_providers {
    oci = { source = "oracle/oci", version = ">= 5.0" }
  }
}

provider "oci" {
  region = var.region
}

variable "tenancy_ocid" {
  description = "OCID da tenancy. No Cloud Shell: export TF_VAR_tenancy_ocid=$OCI_TENANCY"
  type        = string
}

variable "region" {
  type    = string
  default = "sa-saopaulo-1"
}

variable "ssh_public_key" {
  description = "Chave SSH PUBLICA, so para depurar. Vazia = porta 22 fechada."
  type        = string
  default     = ""
}

variable "alert_email" {
  description = "E-mail que recebe alerta se aparecer qualquer custo no mes (1% de US$ 1). Vazio = sem alerta."
  type        = string
  default     = ""
}

variable "online_enabled" {
  description = "Interruptor de emergencia: false desliga o online em todos os apps, sem atualizar o app."
  type        = bool
  default     = true
}

variable "relay_url_expires" {
  description = "Validade do link de leitura do binario no bucket."
  type        = string
  default     = "2030-01-01T00:00:00Z"
}

locals {
  compartment = var.tenancy_ocid
  shape       = "VM.Standard.E2.1.Micro" # Always Free: 1/8 OCPU, 1 GB, ate 50 Mbps
  relay_port  = 9000                     # network.OnlineServer = "<ip>:9000"
  relay_file  = "${path.module}/legiao-relay"
  ssh_rules   = var.ssh_public_key == "" ? [] : [22]
}

# ---------- binario do relay no Object Storage (Always Free: 20 GB) ----------

data "oci_objectstorage_namespace" "ns" {
  compartment_id = var.tenancy_ocid
}

resource "oci_objectstorage_bucket" "relay" {
  compartment_id = local.compartment
  namespace      = data.oci_objectstorage_namespace.ns.namespace
  name           = "legiao-relay"
  access_type    = "NoPublicAccess"
}

resource "oci_objectstorage_object" "relay" {
  namespace    = data.oci_objectstorage_namespace.ns.namespace
  bucket       = oci_objectstorage_bucket.relay.name
  object       = "legiao-relay"
  source       = abspath(local.relay_file)
  content_type = "application/octet-stream"
  # Faz o terraform notar um binario novo com o mesmo nome.
  metadata = { "sha256" = filesha256(local.relay_file) }
}

# Link de leitura so deste objeto: a VM baixa por ele sem credencial nenhuma.
resource "oci_objectstorage_preauthrequest" "relay" {
  namespace    = data.oci_objectstorage_namespace.ns.namespace
  bucket       = oci_objectstorage_bucket.relay.name
  name         = "legiao-relay-read"
  access_type  = "ObjectRead"
  object_name  = oci_objectstorage_object.relay.object
  time_expires = var.relay_url_expires
}

locals {
  relay_url = "https://objectstorage.${var.region}.oraclecloud.com${oci_objectstorage_preauthrequest.relay.access_uri}"
}

# ---------- configuracao publica que o app le ----------
#
# O app NAO tem o IP da VM gravado: ele le este arquivo, num endereco que nao
# muda quando a VM e recriada (regiao + namespace + bucket + nome fixos). E o
# equivalente a um objeto publico no S3. Nao e segredo (o app inteiro precisa
# ler), por isso nao vai no Vault: um cofre exigiria credencial dentro do app.
#
# Bucket so para ele, leitura publica SEM listagem: ninguem ve outros objetos.
# Always Free: 20 GB e 50 mil requisicoes por mes; o app le no maximo uma vez
# a cada 10 minutos de uso e guarda a ultima copia boa.

resource "oci_objectstorage_bucket" "public" {
  compartment_id = local.compartment
  namespace      = data.oci_objectstorage_namespace.ns.namespace
  name           = "legiao-public"
  access_type    = "ObjectReadWithoutList"
}

resource "oci_objectstorage_object" "online_config" {
  namespace     = data.oci_objectstorage_namespace.ns.namespace
  bucket        = oci_objectstorage_bucket.public.name
  object        = "online.json"
  content_type  = "application/json"
  cache_control = "max-age=60"
  content = jsonencode({
    relay   = "${oci_core_instance.relay.public_ip}:${local.relay_port}"
    enabled = var.online_enabled
  })
}

# ---------- rede ----------

data "oci_identity_availability_domains" "ads" {
  compartment_id = var.tenancy_ocid
}

data "oci_core_images" "ubuntu" {
  compartment_id           = local.compartment
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "22.04"
  shape                    = local.shape
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

resource "oci_core_vcn" "legiao" {
  compartment_id = local.compartment
  display_name   = "legiao-vcn"
  cidr_blocks    = ["10.0.0.0/16"]
  dns_label      = "legiao"
}

resource "oci_core_internet_gateway" "legiao" {
  compartment_id = local.compartment
  vcn_id         = oci_core_vcn.legiao.id
  display_name   = "legiao-igw"
  enabled        = true
}

resource "oci_core_route_table" "public" {
  compartment_id = local.compartment
  vcn_id         = oci_core_vcn.legiao.id
  display_name   = "legiao-public-rt"
  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.legiao.id
  }
}

resource "oci_core_security_list" "legiao" {
  compartment_id = local.compartment
  vcn_id         = oci_core_vcn.legiao.id
  display_name   = "legiao-sl"

  egress_security_rules {
    destination = "0.0.0.0/0"
    protocol    = "all"
  }

  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "6" # TCP
    tcp_options {
      min = local.relay_port
      max = local.relay_port
    }
  }

  dynamic "ingress_security_rules" {
    for_each = local.ssh_rules
    content {
      source   = "0.0.0.0/0"
      protocol = "6"
      tcp_options {
        min = ingress_security_rules.value
        max = ingress_security_rules.value
      }
    }
  }
}

resource "oci_core_subnet" "public" {
  compartment_id    = local.compartment
  vcn_id            = oci_core_vcn.legiao.id
  display_name      = "legiao-public"
  cidr_block        = "10.0.1.0/24"
  dns_label         = "pub"
  route_table_id    = oci_core_route_table.public.id
  security_list_ids = [oci_core_security_list.legiao.id]
}

# ---------- VM ----------

resource "oci_core_instance" "relay" {
  compartment_id      = local.compartment
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  display_name        = "legiao-relay"
  shape               = local.shape

  create_vnic_details {
    subnet_id        = oci_core_subnet.public.id
    assign_public_ip = true
  }

  source_details {
    source_type             = "image"
    source_id               = data.oci_core_images.ubuntu.images[0].id
    boot_volume_size_in_gbs = 50
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data = base64encode(<<-EOT
      #!/bin/bash
      set -eux
      # As imagens Ubuntu da Oracle trazem um REJECT no iptables: sem isto a
      # porta fica fechada mesmo com a security list liberada.
      iptables -I INPUT 1 -p tcp --dport ${local.relay_port} -j ACCEPT
      export DEBIAN_FRONTEND=noninteractive
      apt-get update
      apt-get install -y netfilter-persistent iptables-persistent curl
      netfilter-persistent save

      mkdir -p /opt/legiao-relay
      echo '${local.relay_url}' > /opt/legiao-relay/url

      cat > /etc/systemd/system/legiao-relay.service <<'UNIT'
      [Unit]
      Description=Legiao relay
      After=network-online.target
      Wants=network-online.target

      [Service]
      # "+" roda como root: baixa o binario mais novo do bucket. Se o download
      # falhar, o binario anterior continua valendo.
      ExecStartPre=+/bin/sh -c 'curl -fsSL -o /opt/legiao-relay/relay.new "$$(cat /opt/legiao-relay/url)" && install -m 0755 /opt/legiao-relay/relay.new /opt/legiao-relay/relay || true'
      ExecStart=/opt/legiao-relay/relay -listen :${local.relay_port}
      Restart=always
      RestartSec=3
      DynamicUser=yes

      [Install]
      WantedBy=multi-user.target
      UNIT

      systemctl daemon-reload
      systemctl enable --now legiao-relay
    EOT
    )
  }

  # Uma imagem Ubuntu nova, ou um link novo do bucket, NAO pode recriar a VM:
  # recriar muda o IP publico que o jogo tem gravado.
  lifecycle {
    ignore_changes = [source_details, metadata]
  }
}

# ---------- alerta de custo (budgets sao gratuitos) ----------
#
# Tudo neste arquivo e Always Free: o esperado e US$ 0,00 todo mes. O alerta
# existe para avisar se algo sair do gratuito (por exemplo, alguem criar outro
# recurso na conta). Ative com: export TF_VAR_alert_email=seu@email

resource "oci_budget_budget" "guard" {
  count          = var.alert_email == "" ? 0 : 1
  compartment_id = var.tenancy_ocid
  display_name   = "legiao-guarda-custo"
  amount         = 1
  reset_period   = "MONTHLY"
  target_type    = "COMPARTMENT"
  targets        = [var.tenancy_ocid]
}

resource "oci_budget_alert_rule" "any_cost" {
  count          = var.alert_email == "" ? 0 : 1
  budget_id      = oci_budget_budget.guard[0].id
  display_name   = "legiao-qualquer-custo"
  type           = "ACTUAL"
  threshold      = 1
  threshold_type = "PERCENTAGE"
  recipients     = var.alert_email
  message        = "Legiao: apareceu custo na conta Oracle. O relay deveria custar US$ 0."
}

# ---------- saidas ----------

output "relay_address" {
  description = "IP:porta atual do relay (o app le isto de online_config_url)."
  value       = "${oci_core_instance.relay.public_ip}:${local.relay_port}"
}

output "online_config_url" {
  description = "Deve ser igual a network.OnlineConfigURL (internal/network/online_server.go)."
  value       = "https://objectstorage.${var.region}.oraclecloud.com/n/${data.oci_objectstorage_namespace.ns.namespace}/b/${oci_objectstorage_bucket.public.name}/o/${oci_objectstorage_object.online_config.object}"
}

output "health_check" {
  description = "Cole no Cloud Shell para ver a saude do relay."
  value       = "exec 3<>/dev/tcp/${oci_core_instance.relay.public_ip}/${local.relay_port}; echo 'STATS LEGIAO2' >&3; timeout 5 cat <&3"
}

output "update_relay" {
  description = "Depois de enviar um binario novo e rodar terraform apply."
  value       = "oci compute instance action --action SOFTRESET --instance-id ${oci_core_instance.relay.id}"
}
