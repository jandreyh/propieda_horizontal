import React, { useState } from "react";
import { StatusBar } from "expo-status-bar";
import LoginScreen from "./src/screens/LoginScreen";
import HomeScreen from "./src/screens/HomeScreen";

export default function App() {
  const [token, setToken] = useState<string | null>(null);

  const handleLoginSuccess = (authToken: string) => {
    setToken(authToken);
  };

  const handleLogout = () => {
    setToken(null);
  };

  return (
    <>
      <StatusBar style="auto" />
      {token ? (
        <HomeScreen onLogout={handleLogout} />
      ) : (
        <LoginScreen onLoginSuccess={handleLoginSuccess} />
      )}
    </>
  );
}
