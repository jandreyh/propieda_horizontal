export default function PqrsPage() {
  const placeholderItems = [
    { id: "PQR-001", type: "Peticion", subject: "Solicitud de copia de actas", date: "2026-04-27", status: "Abierto" },
    { id: "PQR-002", type: "Queja", subject: "Mal servicio de aseo en lobby", date: "2026-04-22", status: "En proceso" },
    { id: "PQR-003", type: "Reclamo", subject: "Cobro indebido en recibo de marzo", date: "2026-04-15", status: "Cerrado" },
    { id: "PQR-004", type: "Sugerencia", subject: "Implementar sistema de reciclaje", date: "2026-04-10", status: "Cerrado" },
    { id: "PQR-005", type: "Peticion", subject: "Acceso a estados financieros", date: "2026-04-28", status: "Abierto" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">PQRS</h1>
      <p className="mb-6 text-sm text-gray-500">
        Peticiones, quejas, reclamos y sugerencias
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Tipo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Asunto</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Fecha</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.type}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.subject}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.date}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Cerrado"
                        ? "bg-green-100 text-green-700"
                        : item.status === "En proceso"
                          ? "bg-yellow-100 text-yellow-700"
                          : "bg-blue-100 text-blue-700"
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
