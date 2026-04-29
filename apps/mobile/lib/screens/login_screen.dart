import 'package:flutter/material.dart';
import '../api/client.dart';
import 'home_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key, required this.api});

  final ApiClient api;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _identifier =
      TextEditingController(text: 'admin@demo.ph.localhost');
  final _password = TextEditingController(text: 'admin123');
  bool _loading = false;
  String? _error;

  Future<void> _submit() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      await widget.api.login(_identifier.text.trim(), _password.text);
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => HomeScreen(api: widget.api)),
      );
    } on ApiException catch (e) {
      setState(() => _error = e.detail);
    } catch (e) {
      setState(() => _error = 'Error de red: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFF8FAFC),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Container(
                  height: 56,
                  width: 56,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: const Color(0xFF4F46E5),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Text(
                    'PH',
                    style: TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.bold,
                      fontSize: 18,
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Propiedad Horizontal',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.w700,
                    color: Color(0xFF0F172A),
                  ),
                ),
                const SizedBox(height: 4),
                const Text(
                  'App movil del conjunto',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Color(0xFF64748B), fontSize: 14),
                ),
                const SizedBox(height: 28),
                Container(
                  padding: const EdgeInsets.all(20),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: const Color(0xFFE2E8F0)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (_error != null)
                        Container(
                          padding: const EdgeInsets.all(12),
                          margin: const EdgeInsets.only(bottom: 12),
                          decoration: BoxDecoration(
                            color: const Color(0xFFFEF2F2),
                            border: Border.all(color: const Color(0xFFFECACA)),
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Text(
                            _error!,
                            style: const TextStyle(
                                color: Color(0xFFB91C1C), fontSize: 13),
                          ),
                        ),
                      const Text(
                        'Correo o documento',
                        style: TextStyle(
                            fontSize: 13,
                            color: Color(0xFF334155),
                            fontWeight: FontWeight.w500),
                      ),
                      const SizedBox(height: 6),
                      TextField(
                        controller: _identifier,
                        decoration: const InputDecoration(
                          hintText: 'correo@ejemplo.com',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                      const SizedBox(height: 14),
                      const Text(
                        'Contrasena',
                        style: TextStyle(
                            fontSize: 13,
                            color: Color(0xFF334155),
                            fontWeight: FontWeight.w500),
                      ),
                      const SizedBox(height: 6),
                      TextField(
                        controller: _password,
                        obscureText: true,
                        decoration: const InputDecoration(
                          hintText: '********',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                      const SizedBox(height: 20),
                      FilledButton(
                        style: FilledButton.styleFrom(
                          backgroundColor: const Color(0xFF4F46E5),
                          padding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                        onPressed: _loading ? null : _submit,
                        child: Text(
                          _loading ? 'Ingresando...' : 'Iniciar sesion',
                          style: const TextStyle(fontSize: 14),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 12),
                const Text(
                  'Demo: admin@demo.ph.localhost / admin123',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
