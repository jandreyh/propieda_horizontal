import React, { useState } from "react";
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
} from "react-native";
import { post, type LoginResponse, type Membership } from "../lib/api";

interface LoginScreenProps {
  onLoginSuccess: (data: {
    accessToken: string;
    refreshToken?: string;
    memberships: Membership[];
  }) => void;
}

const DOC_TYPES = ["CC", "CE", "PA", "TI", "RC", "NIT"] as const;
type DocType = (typeof DOC_TYPES)[number];

export default function LoginScreen({ onLoginSuccess }: LoginScreenProps) {
  const [email, setEmail] = useState("");
  const [docType, setDocType] = useState<DocType>("CC");
  const [docNumber, setDocNumber] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async () => {
    if (
      !email.trim() ||
      !docNumber.trim() ||
      !password.trim()
    ) {
      setError("Completa correo, documento y contrasena.");
      return;
    }

    setLoading(true);
    setError(null);

    const result = await post<LoginResponse>("/auth/login", {
      email: email.trim().toLowerCase(),
      document_type: docType,
      document_number: docNumber.trim(),
      password,
    });

    setLoading(false);

    if (result.error) {
      setError(result.error);
      return;
    }

    if (result.data?.mfa_required) {
      setError("MFA aun no soportado en mobile. Usa la web por ahora.");
      return;
    }

    if (!result.data?.access_token) {
      setError("Respuesta inesperada del servidor.");
      return;
    }

    onLoginSuccess({
      accessToken: result.data.access_token,
      refreshToken: result.data.refresh_token,
      memberships: result.data.memberships ?? [],
    });
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <View style={styles.form}>
        <Text style={styles.title}>Propiedad Horizontal</Text>
        <Text style={styles.subtitle}>Iniciar Sesion</Text>

        {error ? <Text style={styles.error}>{error}</Text> : null}

        <TextInput
          style={styles.input}
          placeholder="Correo electronico"
          keyboardType="email-address"
          autoCapitalize="none"
          autoCorrect={false}
          value={email}
          onChangeText={setEmail}
        />

        <View style={styles.row}>
          <View style={styles.docTypeBox}>
            <Text style={styles.smallLabel}>Tipo</Text>
            <View style={styles.docTypeChips}>
              {DOC_TYPES.map((t) => (
                <TouchableOpacity
                  key={t}
                  onPress={() => setDocType(t)}
                  style={[
                    styles.chip,
                    docType === t && styles.chipActive,
                  ]}
                >
                  <Text
                    style={[
                      styles.chipText,
                      docType === t && styles.chipTextActive,
                    ]}
                  >
                    {t}
                  </Text>
                </TouchableOpacity>
              ))}
            </View>
          </View>
        </View>

        <TextInput
          style={styles.input}
          placeholder="Numero de documento"
          keyboardType="numeric"
          value={docNumber}
          onChangeText={setDocNumber}
        />

        <TextInput
          style={styles.input}
          placeholder="Contrasena"
          secureTextEntry
          value={password}
          onChangeText={setPassword}
        />

        <TouchableOpacity
          style={[styles.button, loading && styles.buttonDisabled]}
          onPress={handleLogin}
          disabled={loading}
        >
          {loading ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Ingresar</Text>
          )}
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#f5f5f5",
    justifyContent: "center",
    paddingHorizontal: 24,
  },
  form: {
    backgroundColor: "#ffffff",
    borderRadius: 12,
    padding: 24,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 8,
    elevation: 4,
  },
  title: {
    fontSize: 24,
    fontWeight: "700",
    textAlign: "center",
    color: "#1a1a1a",
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 16,
    textAlign: "center",
    color: "#666",
    marginBottom: 24,
  },
  input: {
    borderWidth: 1,
    borderColor: "#ddd",
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 12,
    fontSize: 16,
    marginBottom: 16,
    backgroundColor: "#fafafa",
  },
  row: {
    marginBottom: 16,
  },
  docTypeBox: {
    flex: 1,
  },
  smallLabel: {
    fontSize: 12,
    color: "#666",
    marginBottom: 6,
  },
  docTypeChips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 6,
  },
  chip: {
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 6,
    borderWidth: 1,
    borderColor: "#ccc",
    backgroundColor: "#fafafa",
  },
  chipActive: {
    backgroundColor: "#2563eb",
    borderColor: "#2563eb",
  },
  chipText: {
    fontSize: 13,
    color: "#444",
    fontWeight: "500",
  },
  chipTextActive: {
    color: "#fff",
  },
  button: {
    backgroundColor: "#2563eb",
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: "center",
    marginTop: 8,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: "#ffffff",
    fontSize: 16,
    fontWeight: "600",
  },
  error: {
    color: "#dc2626",
    fontSize: 14,
    textAlign: "center",
    marginBottom: 16,
  },
});
