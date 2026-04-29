export default function AccessControlPage() {
  const placeholderItems = [
    { id: "ACC-001", visitor: "Juan Perez", unit: "Apto 301", type: "Visitante", entry: "2026-04-28 10:15", exit: "--", status: "Dentro" },
    { id: "ACC-002", visitor: "Maria Lopez", unit: "Apto 805", type: "Domicilio", entry: "2026-04-28 09:45", exit: "2026-04-28 09:50", status: "Salio" },
    { id: "ACC-003", visitor: "Carlos Ruiz", unit: "Local 2", type: "Proveedor", entry: "2026-04-28 08:00", exit: "--", status: "Dentro" },
    { id: "ACC-004", visitor: "Ana Garcia", unit: "Apto 1102", type: "Visitante", entry: "2026-04-27 18:30", exit: "2026-04-27 21:00", status: "Salio" },
    { id: "ACC-005", visitor: "Pedro Martinez", unit: "Apto 402", type: "Trabajador", entry: "2026-04-28 07:00", exit: "--", status: "Dentro" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Control de acceso</h1>
      <p className="mb-6 text-sm text-gray-500">
        Registro de ingreso y salida de visitantes y proveedores
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Visitante</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Unidad</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Tipo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Entrada</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Salida</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.visitor}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.unit}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.type}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.entry}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.exit}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Dentro"
                        ? "bg-blue-100 text-blue-700"
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
