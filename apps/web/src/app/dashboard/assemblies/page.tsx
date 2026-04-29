export default function AssembliesPage() {
  const placeholderItems = [
    { id: "ASM-001", title: "Asamblea ordinaria anual 2026", date: "2026-03-15", quorum: "72%", status: "Finalizada" },
    { id: "ASM-002", title: "Asamblea extraordinaria - Fachada", date: "2026-05-10", quorum: "--", status: "Programada" },
    { id: "ASM-003", title: "Asamblea ordinaria anual 2025", date: "2025-03-20", quorum: "68%", status: "Finalizada" },
    { id: "ASM-004", title: "Asamblea extraordinaria - Seguridad", date: "2025-09-05", quorum: "55%", status: "Finalizada" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Asambleas</h1>
      <p className="mb-6 text-sm text-gray-500">
        Reuniones de copropietarios, actas y votaciones
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Titulo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Fecha</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Quorum</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.title}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.date}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.quorum}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Finalizada"
                        ? "bg-green-100 text-green-700"
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
