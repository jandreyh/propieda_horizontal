import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

const String defaultBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8080',
);
const String defaultTenantSlug = String.fromEnvironment(
  'TENANT_SLUG',
  defaultValue: 'demo',
);

const String _kAccess = 'ph.access_token';
const String _kRefresh = 'ph.refresh_token';

class ApiException implements Exception {
  ApiException({
    required this.status,
    required this.title,
    required this.detail,
  });

  final int status;
  final String title;
  final String detail;

  @override
  String toString() => '$status $title — $detail';
}

class ApiClient {
  ApiClient({String? baseUrl, String? tenantSlug})
    : baseUrl = baseUrl ?? defaultBaseUrl,
      tenantSlug = tenantSlug ?? defaultTenantSlug;

  final String baseUrl;
  final String tenantSlug;

  Future<String?> _accessToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_kAccess);
  }

  Future<void> _saveTokens(String access, String refresh) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kAccess, access);
    await prefs.setString(_kRefresh, refresh);
  }

  Future<void> clearTokens() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_kAccess);
    await prefs.remove(_kRefresh);
  }

  Future<bool> isAuthenticated() async {
    final t = await _accessToken();
    return t != null && t.isNotEmpty;
  }

  Future<Map<String, String>> _headers({bool auth = true}) async {
    final h = <String, String>{
      'Content-Type': 'application/json',
      'X-Tenant-Slug': tenantSlug,
    };
    if (auth) {
      final t = await _accessToken();
      if (t != null && t.isNotEmpty) h['Authorization'] = 'Bearer $t';
    }
    return h;
  }

  Future<Map<String, dynamic>> _do(
    String method,
    String path, {
    Object? body,
    bool auth = true,
  }) async {
    final uri = Uri.parse('$baseUrl$path');
    final headers = await _headers(auth: auth);
    final encoded = body == null ? null : jsonEncode(body);

    http.Response res;
    switch (method) {
      case 'GET':
        res = await http.get(uri, headers: headers);
        break;
      case 'POST':
        res = await http.post(uri, headers: headers, body: encoded);
        break;
      case 'DELETE':
        res = await http.delete(uri, headers: headers);
        break;
      default:
        throw UnsupportedError('method $method');
    }

    if (res.statusCode >= 400) {
      Map<String, dynamic> p;
      try {
        p = jsonDecode(res.body) as Map<String, dynamic>;
      } catch (_) {
        p = {
          'status': res.statusCode,
          'title': res.reasonPhrase ?? 'Error',
          'detail': res.reasonPhrase ?? 'Error',
        };
      }
      throw ApiException(
        status: (p['status'] as num?)?.toInt() ?? res.statusCode,
        title: p['title']?.toString() ?? 'Error',
        detail: p['detail']?.toString() ?? p['title']?.toString() ?? 'Error',
      );
    }

    if (res.body.isEmpty) return <String, dynamic>{};
    final decoded = jsonDecode(res.body);
    if (decoded is Map<String, dynamic>) return decoded;
    return {'_root': decoded};
  }

  Future<void> login(String identifier, String password) async {
    final res = await _do(
      'POST',
      '/auth/login',
      body: {'identifier': identifier, 'password': password},
      auth: false,
    );
    if (res['mfa_required'] == true) {
      throw ApiException(
        status: 401,
        title: 'MFA requerido',
        detail: 'Esta UI demo no soporta MFA',
      );
    }
    final at = res['access_token']?.toString();
    final rt = res['refresh_token']?.toString();
    if (at == null || rt == null) {
      throw ApiException(
        status: 502,
        title: 'Bad gateway',
        detail: 'respuesta inesperada del API',
      );
    }
    await _saveTokens(at, rt);
  }

  Future<void> logout() async {
    try {
      await _do('POST', '/auth/logout');
    } catch (_) {}
    await clearTokens();
  }

  Future<Map<String, dynamic>> me() => _do('GET', '/me');
  Future<Map<String, dynamic>> packages() =>
      _do('GET', '/packages?status=received');
  Future<Map<String, dynamic>> announcementsFeed() =>
      _do('GET', '/announcements/feed');
  Future<Map<String, dynamic>> activeVisits() => _do('GET', '/visits/active');
}
