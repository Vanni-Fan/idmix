Imports System.IO
Imports System.Numerics
Imports System.Text

Namespace Vanni.Idmix

''' <summary>XID 文本层：自定义进制编解码。</summary>
Public NotInheritable Class RadixCodec
    Private ReadOnly _base As Integer
    Private ReadOnly _chars As String
    Private ReadOnly _fromCustom As Dictionary(Of Char, Integer)

    Public Sub New(alphabet As String)
        If alphabet.Length < 2 Then
            Throw New ArgumentException("alphabet must have at least 2 unique characters", NameOf(alphabet))
        End If
        _base = alphabet.Length
        _chars = alphabet
        _fromCustom = New Dictionary(Of Char, Integer)()
        For i = 0 To alphabet.Length - 1
            Dim c = alphabet(i)
            If _fromCustom.ContainsKey(c) Then
                Throw New ArgumentException($"alphabet contains duplicate character {c}", NameOf(alphabet))
            End If
            _fromCustom(c) = i
        Next
    End Sub

    Public ReadOnly Property Base As Integer
        Get
            Return _base
        End Get
    End Property

    Public ReadOnly Property Chars As String
        Get
            Return _chars
        End Get
    End Property

    Public Function EncodeBytes(data As Byte()) As String
        If data.Length = 0 Then Return _chars(0).ToString()
        Dim buf(1 + data.Length) As Byte
        buf(0) = CByte((data.Length >> 8) And &HFF)
        buf(1) = CByte(data.Length And &HFF)
        Buffer.BlockCopy(data, 0, buf, 2, data.Length)
        Dim bytes = CType(buf.Clone(), Byte())
        If BitConverter.IsLittleEndian Then Array.Reverse(bytes)
        Dim n As New BigInteger(bytes)
        Return IntToString(n)
    End Function

    Public Function DecodeBytes(s As String) As Byte()
        If String.IsNullOrEmpty(s) Then Throw New ArgumentException("empty string", NameOf(s))
        Dim n = StringToInt(s)
        Dim raw = n.ToByteArray()
        If raw.Length > 0 AndAlso raw(raw.Length - 1) = 0 Then
            ReDim Preserve raw(raw.Length - 2)
        End If
        Array.Reverse(raw)
        For pad = 0 To 1
            Dim bufLen = pad + raw.Length
            Dim buf(bufLen - 1) As Byte
            If pad > 0 Then buf(0) = 0
            Buffer.BlockCopy(raw, 0, buf, pad, raw.Length)
            If buf.Length < 2 Then Continue For
            Dim dataLen = (buf(0) << 8) Or buf(1)
            If buf.Length <> 2 + dataLen Then Continue For
            Dim outBuf(dataLen - 1) As Byte
            Buffer.BlockCopy(buf, 2, outBuf, 0, dataLen)
            Return outBuf
        Next
        Throw New ArgumentException("invalid encoded data length")
    End Function

    Private Function IntToString(n As BigInteger) As String
        If n.IsZero Then Return _chars(0).ToString()
        Dim baseBi As New BigInteger(_base)
        Dim sb As New StringBuilder()
        While n > BigInteger.Zero
            Dim remVal As BigInteger
            n = BigInteger.DivRem(n, baseBi, remVal)
            sb.Append(_chars(CInt(remVal)))
        End While
        Dim chars = sb.ToString().ToCharArray()
        Array.Reverse(chars)
        Return New String(chars)
    End Function

    Private Function StringToInt(s As String) As BigInteger
        Dim n = BigInteger.Zero
        Dim baseBi As New BigInteger(_base)
        For Each c In s
            If Not _fromCustom.ContainsKey(c) Then
                Throw New ArgumentException($"invalid character {c}", NameOf(s))
            End If
            n = n * baseBi + _fromCustom(c)
        Next
        Return n
    End Function
End Class

End Namespace
