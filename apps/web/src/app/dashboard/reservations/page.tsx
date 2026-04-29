export default function ReservationsPage() {
  const placeholderItems = [
    { id: "RES-001", area: "Salon comunal", date: "2026-05-02", time: "14:00 - 18:00", status: "Confirmada" },
    { id: "RES-002", area: "Cancha de futbol", date: "2026-05-03", time: "09:00 - 11:00", status: "Pendiente" },
    { id: "RES-003", area: "BBQ zona norte", date: "2026-05-05", time: "12:00 - 16:00", status: "Confirmada" },
    { id: "RES-004", area: "Salon comunal", date: "2026-05-07", time: "10:00 - 14:00", status: "Cancelada" },
    { id: "RES-005", area: "Piscina (carril 1)", date: "2026-05-08", time: "06:00 - 08:00", status: "Pendiente" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Reservas</h1>
      <p className="mb-6 text-sm text-gray-500">
        Reserva de areas comunes y zonas sociales
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Area</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Fecha</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Horario</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.area}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.date}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.time}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Confirmada"
                        ? "bg-green-100 text-green-700"
                        : item.status === "Pendiente"
                          ? "bg-yellow-100 text-yellow-700"
                          : "bg-red-100 text-red-700"
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
