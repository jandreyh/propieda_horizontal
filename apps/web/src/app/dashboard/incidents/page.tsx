export default function IncidentsPage() {
  const placeholderItems = [
    { id: "INC-001", title: "Fuga de agua piso 3", category: "Infraestructura", date: "2026-04-25", status: "Abierto" },
    { id: "INC-002", title: "Ascensor fuera de servicio", category: "Infraestructura", date: "2026-04-20", status: "En proceso" },
    { id: "INC-003", title: "Ruido excesivo Apto 501", category: "Convivencia", date: "2026-04-18", status: "Cerrado" },
    { id: "INC-004", title: "Vidrio roto lobby principal", category: "Infraestructura", date: "2026-04-15", status: "Cerrado" },
    { id: "INC-005", title: "Mascota sin correa en zona comun", category: "Convivencia", date: "2026-04-28", status: "Abierto" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Incidentes</h1>
      <p className="mb-6 text-sm text-gray-500">
        Reporte y seguimiento de incidentes en el conjunto
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Titulo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Categoria</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Fecha</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.title}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.category}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.date}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Cerrado"
                        ? "bg-green-100 text-green-700"
                        : item.status === "En proceso"
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
