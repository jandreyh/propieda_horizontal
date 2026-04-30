import React, { useState } from "react";
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  FlatList,
  ActivityIndicator,
} from "react-native";
import {
  post,
  type Membership,
  type SwitchTenantResponse,
} from "../lib/api";

interface Props {
  memberships: Membership[];
  onTenantSelected: (slug: string, accessToken: string) => void;
  onLogout: () => void;
}

export default function SelectTenantScreen({
  memberships,
  onTenantSelected,
  onLogout,
}: Props) {
  const [busySlug, setBusySlug] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSelect = async (slug: string) => {
    setBusySlug(slug);
    setError(null);
    const res = await post<SwitchTenantResponse>("/auth/switch-tenant", {
      tenant_slug: slug,
    });
    setBusySlug(null);
    if (res.error || !res.data) {
      setError(res.error || "Error al seleccionar conjunto");
      return;
    }
    onTenantSelected(slug, res.data.access_token);
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Selecciona un conjunto</Text>
      <Text style={styles.subtitle}>
        Tienes {memberships.length}{" "}
        {memberships.length === 1 ? "conjunto" : "conjuntos"} disponibles
      </Text>

      {error ? <Text style={styles.error}>{error}</Text> : null}

      {memberships.length === 0 ? (
        <View style={styles.empty}>
          <Text style={styles.emptyText}>
            No tienes membresias activas. Contacta al administrador para que te
            vincule por tu codigo unico.
          </Text>
        </View>
      ) : (
        <FlatList
          data={memberships}
          keyExtractor={(m) => m.tenant_id}
          contentContainerStyle={styles.list}
          renderItem={({ item }) => (
            <TouchableOpacity
              style={[
                styles.card,
                item.primary_color && {
                  borderLeftColor: item.primary_color,
                  borderLeftWidth: 4,
                },
              ]}
              onPress={() => handleSelect(item.tenant_slug)}
              disabled={busySlug !== null}
            >
              <View style={styles.cardBody}>
                <Text style={styles.cardTitle}>{item.tenant_name}</Text>
                <Text style={styles.cardMeta}>
                  {item.role} · {item.tenant_slug}
                </Text>
              </View>
              {busySlug === item.tenant_slug ? (
                <ActivityIndicator color="#2563eb" />
              ) : (
                <Text style={styles.cardArrow}>→</Text>
              )}
            </TouchableOpacity>
          )}
        />
      )}

      <TouchableOpacity style={styles.logoutButton} onPress={onLogout}>
        <Text style={styles.logoutText}>Cerrar sesion</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#f5f5f5",
    paddingHorizontal: 24,
    paddingTop: 64,
  },
  title: {
    fontSize: 22,
    fontWeight: "700",
    color: "#1a1a1a",
    textAlign: "center",
  },
  subtitle: {
    fontSize: 14,
    color: "#666",
    textAlign: "center",
    marginBottom: 24,
    marginTop: 4,
  },
  list: {
    gap: 12,
    paddingBottom: 16,
  },
  card: {
    backgroundColor: "#fff",
    borderRadius: 10,
    padding: 16,
    flexDirection: "row",
    alignItems: "center",
    shadowColor: "#000",
    shadowOpacity: 0.05,
    shadowRadius: 4,
    shadowOffset: { width: 0, height: 1 },
    elevation: 1,
  },
  cardBody: {
    flex: 1,
  },
  cardTitle: {
    fontSize: 16,
    fontWeight: "600",
    color: "#1a1a1a",
  },
  cardMeta: {
    fontSize: 13,
    color: "#666",
    marginTop: 2,
  },
  cardArrow: {
    fontSize: 20,
    color: "#9CA3AF",
  },
  empty: {
    backgroundColor: "#FEF3C7",
    borderRadius: 8,
    padding: 16,
  },
  emptyText: {
    color: "#92400E",
    textAlign: "center",
  },
  error: {
    color: "#dc2626",
    textAlign: "center",
    marginBottom: 12,
  },
  logoutButton: {
    marginTop: "auto",
    paddingVertical: 16,
    alignItems: "center",
  },
  logoutText: {
    color: "#666",
    fontSize: 14,
  },
});
