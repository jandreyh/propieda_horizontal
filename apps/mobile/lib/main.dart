import 'package:flutter/material.dart';
import 'api/client.dart';
import 'screens/home_screen.dart';
import 'screens/login_screen.dart';

void main() {
  runApp(const PhApp());
}

class PhApp extends StatelessWidget {
  const PhApp({super.key});

  @override
  Widget build(BuildContext context) {
    final api = ApiClient();
    return MaterialApp(
      title: 'Propiedad Horizontal',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF4F46E5),
        ),
        scaffoldBackgroundColor: const Color(0xFFF8FAFC),
        useMaterial3: true,
        appBarTheme: const AppBarTheme(
          elevation: 0,
          centerTitle: false,
          backgroundColor: Colors.white,
          foregroundColor: Color(0xFF0F172A),
        ),
      ),
      home: FutureBuilder<bool>(
        future: api.isAuthenticated(),
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            );
          }
          if (snap.data == true) return HomeScreen(api: api);
          return LoginScreen(api: api);
        },
      ),
    );
  }
}
