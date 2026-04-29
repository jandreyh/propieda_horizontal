import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import '../api/client.dart';
import 'login_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key, required this.api});

  final ApiClient api;

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  Map<String, dynamic>? _me;
  List<dynamic> _packages = [];
  List<dynamic> _announcements = [];
  List<dynamic> _visits = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    Map<String, dynamic>? meRes;
    List<dynamic> packs = [];
    List<dynamic> anns = [];
    List<dynamic> vis = [];
    String? err;

    try {
      meRes = await widget.api.me();
    } catch (e) {
      err = 'No se pudo cargar el perfil ($e)';
    }

    Future<List<dynamic>> safeList(
        Future<Map<String, dynamic>> Function() f) async {
      try {
        final r = await f();
        return (r['items'] as List?) ?? [];
      } catch (_) {
        return [];
      }
    }

    final results = await Future.wait([
      safeList(widget.api.packages),
      safeList(widget.api.announcementsFeed),
      safeList(widget.api.activeVisits),
    ]);
    packs = results[0];
    anns = results[1];
    vis = results[2];

    if (!mounted) return;
    setState(() {
      _me = meRes;
      _packages = packs;
      _announcements = anns;
      _visits = vis;
      _loading = false;
      _error = err;
    });
  }

  Future<void> _logout() async {
    await widget.api.logout();
    if (!mounted) return;
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => LoginScreen(api: widget.api)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final fullName = _me == null
        ? ''
        : '${_me!['names'] ?? ''} ${_me!['last_names'] ?? ''}'.trim();

    return Scaffold(
      backgroundColor: const Color(0xFFF8FAFC),
      appBar: AppBar(
        title: const Text('Propiedad Horizontal'),
        backgroundColor: Colors.white,
        foregroundColor: const Color(0xFF0F172A),
        actions: [
          IconButton(
            tooltip: 'Refrescar',
            icon: const Icon(Icons.refresh),
            onPressed: _loading ? null : _load,
          ),
          IconButton(
            tooltip: 'Cerrar sesion',
            icon: const Icon(Icons.logout),
            onPressed: _logout,
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  if (fullName.isNotEmpty)
                    Text(
                      'Hola, $fullName',
                      style: const TextStyle(
                          fontSize: 18, fontWeight: FontWeight.w600),
                    ),
                  const SizedBox(height: 4),
                  const Text(
                    'Conjunto demo',
                    style: TextStyle(color: Color(0xFF64748B)),
                  ),
                  if (_error != null)
                    Padding(
                      padding: const EdgeInsets.only(top: 12),
                      child: Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: const Color(0xFFFEF2F2),
                          border: Border.all(color: const Color(0xFFFECACA)),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(_error!,
                            style: const TextStyle(color: Color(0xFFB91C1C))),
                      ),
                    ),
                  const SizedBox(height: 20),
                  _Section(
                    title: 'Paquetes en porteria',
                    count: _packages.length,
                    child: _packages.isEmpty
                        ? const _Empty(text: 'Sin paquetes pendientes')
                        : Column(
                            children: _packages.map((p) {
                              final m = p as Map<String, dynamic>;
                              return _ListTile(
                                title: m['recipient_name']?.toString() ?? '—',
                                subtitle:
                                    'Recibido ${_fmt(m['received_at'])} · ${m['carrier'] ?? '—'}',
                                trailing: m['status']?.toString() ?? '',
                                trailingColor: const Color(0xFFD97706),
                              );
                            }).toList(),
                          ),
                  ),
                  const SizedBox(height: 16),
                  _Section(
                    title: 'Visitas activas',
                    count: _visits.length,
                    child: _visits.isEmpty
                        ? const _Empty(text: 'Sin visitas activas')
                        : Column(
                            children: _visits.map((v) {
                              final m = v as Map<String, dynamic>;
                              return _ListTile(
                                title:
                                    m['visitor_full_name']?.toString() ?? '—',
                                subtitle:
                                    '${m['visitor_document_type'] ?? ''} ${m['visitor_document_number'] ?? ''}',
                                trailing: _fmt(m['entry_time']),
                              );
                            }).toList(),
                          ),
                  ),
                  const SizedBox(height: 16),
                  _Section(
                    title: 'Anuncios recientes',
                    count: _announcements.length,
                    child: _announcements.isEmpty
                        ? const _Empty(text: 'Sin anuncios')
                        : Column(
                            children: _announcements.map((a) {
                              final m = a as Map<String, dynamic>;
                              return Container(
                                margin: const EdgeInsets.only(bottom: 10),
                                padding: const EdgeInsets.all(12),
                                decoration: BoxDecoration(
                                  color: Colors.white,
                                  borderRadius: BorderRadius.circular(10),
                                  border: Border.all(
                                      color: const Color(0xFFE2E8F0)),
                                ),
                                child: Column(
                                  crossAxisAlignment:
                                      CrossAxisAlignment.start,
                                  children: [
                                    Row(
                                      children: [
                                        if (m['pinned'] == true)
                                          Container(
                                            padding: const EdgeInsets.symmetric(
                                                horizontal: 6, vertical: 2),
                                            margin:
                                                const EdgeInsets.only(right: 6),
                                            decoration: BoxDecoration(
                                              color:
                                                  const Color(0xFFFFFBEB),
                                              borderRadius:
                                                  BorderRadius.circular(999),
                                            ),
                                            child: const Text(
                                              'Fijado',
                                              style: TextStyle(
                                                  color: Color(0xFFB45309),
                                                  fontSize: 11),
                                            ),
                                          ),
                                        Expanded(
                                          child: Text(
                                            m['title']?.toString() ?? '—',
                                            style: const TextStyle(
                                                fontWeight: FontWeight.w600),
                                          ),
                                        ),
                                      ],
                                    ),
                                    const SizedBox(height: 4),
                                    Text(
                                      m['body']?.toString() ?? '',
                                      style: const TextStyle(
                                          fontSize: 13,
                                          color: Color(0xFF475569)),
                                    ),
                                    const SizedBox(height: 6),
                                    Text(
                                      _fmt(m['published_at']),
                                      style: const TextStyle(
                                          fontSize: 11,
                                          color: Color(0xFF94A3B8)),
                                    ),
                                  ],
                                ),
                              );
                            }).toList(),
                          ),
                  ),
                ],
              ),
            ),
    );
  }
}

String _fmt(dynamic iso) {
  if (iso == null) return '—';
  try {
    final d = DateTime.parse(iso.toString()).toLocal();
    return DateFormat('dd/MM/yyyy HH:mm').format(d);
  } catch (_) {
    return iso.toString();
  }
}

class _Section extends StatelessWidget {
  const _Section({
    required this.title,
    required this.count,
    required this.child,
  });

  final String title;
  final int count;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFFE2E8F0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  title,
                  style: const TextStyle(
                      fontSize: 14, fontWeight: FontWeight.w600),
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(
                    horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: const Color(0xFFEEF2FF),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  '$count',
                  style: const TextStyle(
                      color: Color(0xFF4338CA),
                      fontSize: 12,
                      fontWeight: FontWeight.w600),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _Empty extends StatelessWidget {
  const _Empty({required this.text});
  final String text;
  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Text(
          text,
          textAlign: TextAlign.center,
          style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 13),
        ),
      );
}

class _ListTile extends StatelessWidget {
  const _ListTile({
    required this.title,
    required this.subtitle,
    this.trailing,
    this.trailingColor,
  });

  final String title;
  final String subtitle;
  final String? trailing;
  final Color? trailingColor;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: const TextStyle(
                        fontSize: 14, fontWeight: FontWeight.w500)),
                Text(
                  subtitle,
                  style: const TextStyle(
                      fontSize: 12, color: Color(0xFF94A3B8)),
                ),
              ],
            ),
          ),
          if (trailing != null)
            Container(
              padding: const EdgeInsets.symmetric(
                  horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: (trailingColor ?? const Color(0xFF64748B))
                    .withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                trailing!,
                style: TextStyle(
                  color: trailingColor ?? const Color(0xFF334155),
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
        ],
      ),
    );
  }
}
