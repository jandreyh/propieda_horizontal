export default function PenaltiesPage() {
  const placeholderItems = [
    { id: "SAN-001", unit: "Apto 501", reason: "Ruido excesivo en horario nocturno", amount: "$200,000", date: "2026-04-18", status: "Vigente" },
    { id: "SAN-002", unit: "Apto 302", reason: "Mascota sin correa en zona comun", amount: "$100,000", date: "2026-04-10", status: "Pagada" },
    { id: "SAN-003", unit: "Local 3", reason: "Uso indebido de parqueadero", amount: "$150,000", date: "2026-03-25", status: "Apelada" },
    { id: "SAN-004", unit: "Apto 1201", reason: "Dano a propiedad comun", amount: "$500,000", date: "2026-03-15", status: "Vigente" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Sanciones</h1>
      <p className="mb-6 text-sm text-gray-500">
        Multas y sanciones aplicadas a unidades infractoras
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Unidad</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Motivo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Monto</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Fecha</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.unit}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.reason}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.amount}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.date}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Pagada"
                        ? "bg-green-100 text-green-700"
                        : item.status === "Apelada"
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
