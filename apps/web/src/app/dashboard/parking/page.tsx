export default function ParkingPage() {
  const placeholderItems = [
    { id: "PRK-001", space: "P-101", unit: "Apto 301", vehicle: "ABC-123", type: "Fijo", status: "Ocupado" },
    { id: "PRK-002", space: "P-102", unit: "Apto 502", vehicle: "DEF-456", type: "Fijo", status: "Ocupado" },
    { id: "PRK-003", space: "P-201", unit: "--", vehicle: "--", type: "Visitante", status: "Disponible" },
    { id: "PRK-004", space: "P-202", unit: "Apto 801", vehicle: "GHI-789", type: "Fijo", status: "Ocupado" },
    { id: "PRK-005", space: "P-203", unit: "--", vehicle: "--", type: "Visitante", status: "Disponible" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Parqueaderos</h1>
      <p className="mb-6 text-sm text-gray-500">
        Gestion de espacios de parqueo y asignaciones
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Espacio</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Unidad</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Vehiculo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Tipo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.space}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.unit}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.vehicle}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.type}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Disponible"
                        ? "bg-green-100 text-green-700"
                        : "bg-gray-100 text-gray-700"
                    }`}
                  >
                    {item.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
