import React from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  SafeAreaView,
} from "react-native";

interface HomeScreenProps {
  onLogout: () => void;
}

interface ModuleCard {
  id: string;
  name: string;
  description: string;
}

const MODULES: ModuleCard[] = [
  { id: "finanzas", name: "Finanzas", description: "Cuotas, pagos y estados de cuenta" },
  { id: "reservas", name: "Reservas", description: "Zonas comunes y salones" },
  { id: "asambleas", name: "Asambleas", description: "Convocatorias y votaciones" },
  { id: "incidentes", name: "Incidentes", description: "Reportes y seguimiento" },
  { id: "multas", name: "Multas", description: "Infracciones y sanciones" },
  { id: "pqrs", name: "PQRS", description: "Peticiones, quejas, reclamos y sugerencias" },
  { id: "notificaciones", name: "Notificaciones", description: "Avisos y alertas" },
  { id: "parqueaderos", name: "Parqueaderos", description: "Asignacion y control vehicular" },
  { id: "paqueteria", name: "Paqueteria", description: "Correspondencia y paquetes" },
  { id: "control_acceso", name: "Control de Acceso", description: "Visitantes y porteria" },
  { id: "anuncios", name: "Anuncios", description: "Tablero de anuncios del conjunto" },
];

function ModuleItem({ item }: { item: ModuleCard }) {
  return (
    <TouchableOpacity style={styles.card} activeOpacity={0.7}>
      <Text style={styles.cardTitle}>{item.name}</Text>
      <Text style={styles.cardDescription}>{item.description}</Text>
    </TouchableOpacity>
  );
}

export default function HomeScreen({ onLogout }: HomeScreenProps) {
  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Propiedad Horizontal</Text>
        <TouchableOpacity onPress={onLogout} style={styles.logoutButton}>
          <Text style={styles.logoutText}>Salir</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={MODULES}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <ModuleItem item={item} />}
        contentContainerStyle={styles.list}
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#f5f5f5",
  },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingHorizontal: 16,
    paddingVertical: 16,
    backgroundColor: "#2563eb",
  },
  headerTitle: {
    fontSize: 20,
    fontWeight: "700",
    color: "#ffffff",
  },
  logoutButton: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    backgroundColor: "rgba(255,255,255,0.2)",
    borderRadius: 6,
  },
  logoutText: {
    color: "#ffffff",
    fontSize: 14,
    fontWeight: "600",
  },
  list: {
    padding: 16,
    paddingBottom: 32,
  },
  card: {
    backgroundColor: "#ffffff",
    borderRadius: 10,
    padding: 16,
    marginBottom: 12,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08,
    shadowRadius: 4,
    elevation: 2,
  },
  cardTitle: {
    fontSize: 17,
    fontWeight: "600",
    color: "#1a1a1a",
    marginBottom: 4,
  },
  cardDescription: {
    fontSize: 14,
    color: "#666",
  },
});
