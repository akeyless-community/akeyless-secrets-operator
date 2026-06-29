#!/usr/bin/env python3
"""Post-process Helm CRD templates split from bundle.yaml."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path


def yq_query(path: Path, expr: str) -> str:
    result = subprocess.run(
        ["yq", "e", expr, str(path)],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def guard_for_kind(kind: str) -> str:
    flag = f"create{kind}"
    if "Akeyless" in kind or "Cluster" in kind or "PushSecret" in kind:
        return "{{- if and (.Values.installCRDs) (.Values.crds." + flag + ") }}"
    if flag in ("createExternalSecret", "createSecretStore"):
        return "{{- if and (.Values.installCRDs) (.Values.crds." + flag + ") }}"
    return "{{- if and (.Values.installCRDs) (.Values.crds.createGenerators) }}"


def legacy_external_secret_kinds() -> set[str]:
    return {
        "ExternalSecret",
        "ClusterExternalSecret",
        "SecretStore",
        "ClusterSecretStore",
    }


def process_external_secret_store(content: str, kind: str) -> str:
    if kind not in legacy_external_secret_kinds():
        return content

    content = re.sub(
        r"^      served: false$\n      storage: false",
        "      served: {{ .Values.crds.unsafeServeV1Beta1 }}\n      storage: false",
        content,
        flags=re.MULTILINE,
    )
    content = re.sub(r"^\s*- \|-\n", "", content, flags=re.MULTILINE)
    content = content.replace(
        "       additionalPrinterColumns:",
        "    - additionalPrinterColumns:",
    )
    return content


def inject_annotations(content: str) -> str:
    marker = "  annotations:"
    idx = content.find(marker)
    if idx == -1:
        return content
    insert_at = idx + len(marker)
    injection = (
        "\n    {{- with .Values.crds.annotations }}"
        "\n    {{- toYaml . | nindent 4}}"
        "\n    {{- end }}"
        "\n    {{- if and .Values.crds.conversion.enabled .Values.webhook.certManager.enabled .Values.webhook.certManager.addInjectorAnnotations }}"
        "\n    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include \"external-secrets.fullname\" . }}-webhook"
        "\n    {{- end }}"
    )
    return content[:insert_at] + injection + content[insert_at:]


def process_file(path: Path) -> None:
    kind = yq_query(path, ".spec.names.kind")
    body = path.read_text()
    if kind in legacy_external_secret_kinds():
        body = process_external_secret_store(body, kind)

    body = body.replace(
        "name: kubernetes",
        'name: {{ include "external-secrets.fullname" . }}-webhook',
    )
    body = body.replace(
        "namespace: default",
        "namespace: {{ .Release.Namespace | quote }}",
    )
    body = inject_annotations(body)

    wrapped = guard_for_kind(kind) + "\n" + body + "\n{{- end }}\n"
    out_path = path.with_suffix(".yaml")
    out_path.write_text(wrapped)
    path.unlink(missing_ok=True)


def main() -> int:
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} <helm-crds-templates-dir>", file=sys.stderr)
        return 1

    crds_dir = Path(sys.argv[1])
    for path in sorted(crds_dir.glob("*.yml")):
        process_file(path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
