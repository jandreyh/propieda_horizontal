import React, { useState } from "react";
import { StatusBar } from "expo-status-bar";

import LoginScreen from "./src/screens/LoginScreen";
import SelectTenantScreen from "./src/screens/SelectTenantScreen";
import HomeScreen from "./src/screens/HomeScreen";
import { setBearer, type Membership } from "./src/lib/api";

type AuthState =
  | { stage: "login" }
  | { stage: "select"; memberships: Membership[] }
  | { stage: "home"; tenantSlug: string; memberships: Membership[] };

export default function App() {
  const [auth, setAuth] = useState<AuthState>({ stage: "login" });

  const handleLoginSuccess = (data: {
    accessToken: string;
    refreshToken?: string;
    memberships: Membership[];
  }) => {
    setBearer(data.accessToken);
    if (data.memberships.length === 1) {
      // El cliente puede entrar directo, pero sigue siendo buena idea
      // pasar por SelectTenantScreen para ejercitar el flow. Saltamos
      // si hay un solo conjunto activo.
      setAuth({ stage: "select", memberships: data.memberships });
    } else {
      setAuth({ stage: "select", memberships: data.memberships });
    }
  };

  const handleTenantSelected = (slug: string, accessToken: string) => {
    setBearer(accessToken);
    setAuth((prev) =>
      prev.stage === "select"
        ? { stage: "home", tenantSlug: slug, memberships: prev.memberships }
        : prev,
    );
  };

  const handleLogout = () => {
    setBearer(null);
    setAuth({ stage: "login" });
  };

  return (
    <>
      <StatusBar style="auto" />
      {auth.stage === "login" && (
        <LoginScreen onLoginSuccess={handleLoginSuccess} />
      )}
      {auth.stage === "select" && (
        <SelectTenantScreen
          memberships={auth.memberships}
          onTenantSelected={handleTenantSelected}
          onLogout={handleLogout}
        />
      )}
      {auth.stage === "home" && <HomeScreen onLogout={handleLogout} />}
    </>
  );
}
