export default function PackagesPage() {
  const placeholderItems = [
    { id: "PKG-001", unit: "Apto 301", sender: "Amazon", received: "2026-04-28 09:30", type: "Paquete", status: "En porteria" },
    { id: "PKG-002", unit: "Apto 805", sender: "MercadoLibre", received: "2026-04-27 14:20", type: "Paquete", status: "Entregado" },
    { id: "PKG-003", unit: "Apto 1102", sender: "Banco Davivienda", received: "2026-04-27 10:00", type: "Correspondencia", status: "En porteria" },
    { id: "PKG-004", unit: "Local 2", sender: "Servientrega", received: "2026-04-26 16:45", type: "Paquete", status: "Entregado" },
    { id: "PKG-005", unit: "Apto 402", sender: "Rappi", received: "2026-04-28 12:10", type: "Domicilio", status: "Entregado" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Paquetes</h1>
      <p className="mb-6 text-sm text-gray-500">
        Correspondencia, paqueteria y domicilios
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Unidad</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Remitente</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Recibido</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Tipo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.unit}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.sender}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.received}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.type}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Entregado"
                        ? "bg-green-100 text-green-700"
                        : "bg-yellow-100 text-yellow-700"
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
