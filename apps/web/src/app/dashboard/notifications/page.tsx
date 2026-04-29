export default function NotificationsPage() {
  const placeholderItems = [
    { id: "NOT-001", title: "Pago recibido exitosamente", channel: "Email", date: "2026-04-28 10:30", read: true },
    { id: "NOT-002", title: "Asamblea programada para el 10 de mayo", channel: "Push", date: "2026-04-27 14:00", read: false },
    { id: "NOT-003", title: "Nuevo paquete en porteria", channel: "Email", date: "2026-04-26 09:15", read: true },
    { id: "NOT-004", title: "Su PQRS ha sido respondida", channel: "Push", date: "2026-04-25 16:45", read: false },
    { id: "NOT-005", title: "Recordatorio: cuota vence manana", channel: "SMS", date: "2026-04-24 08:00", read: true },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Notificaciones</h1>
      <p className="mb-6 text-sm text-gray-500">
        Alertas, avisos y preferencias de comunicacion
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Titulo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Canal</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Fecha</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Leida</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className={`hover:bg-gray-50 ${!item.read ? "bg-blue-50" : ""}`}>
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.title}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.channel}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.date}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.read ? "Si" : "No"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
