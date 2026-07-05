Imports System.IO
Imports System.Numerics

Namespace Vanni.Idmix

''' <summary>XID v1.1 编解码器主入口。</summary>
Public NotInheritable Class IdMix
    Public Const DefaultAlphabet As String =
        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

    Private ReadOnly _radix As RadixCodec
    Private ReadOnly _random As New Random()

    Public ReadOnly Property MaxObjects As Integer
    Public ReadOnly Property MaxVariants As Integer
    Public ReadOnly Property CheckBits As Integer
    Public Property CountBits As Integer
    Public Property VariantBits As Integer
    Public Property CheckMask As Integer
    Public Property CountMask As Integer
    Public Property VariantMask As Integer
    Public Property CountShift As Integer
    Public Property VariantShift As Integer

    Public Sub New()
        Me.New(DefaultAlphabet, 511, 32, 2)
    End Sub

    Public Sub New(alphabet As String)
        Me.New(alphabet, 511, 32, 2)
    End Sub

    Public Sub New(alphabet As String, maxObjects As Integer, maxVariants As Integer, checkBits As Integer)
        _radix = New RadixCodec(alphabet)
        Me.MaxObjects = maxObjects
        Me.MaxVariants = maxVariants
        Me.CheckBits = checkBits
        FinalizeLayout()
    End Sub

    Public Shared Function NewDefault() As IdMix
        Return New IdMix()
    End Function

    Public ReadOnly Property Radix As RadixCodec
        Get
            Return _radix
        End Get
    End Property

    Public Function Encode(ParamArray values As TypedValue()) As String
        If values.Length < 1 Then Throw New ArgumentException("at least one value is required")
        If values.Length > MaxObjects Then Throw New ArgumentException($"too many objects: {values.Length}")
        Dim variantId = _random.Next(MaxVariants)
        Dim data = XidCodec.EncodeBinary(Me, values, variantId)
        Return _radix.EncodeBytes(data)
    End Function

    Public Function Decode(s As String) As List(Of TypedValue)
        Dim data = _radix.DecodeBytes(s)
        Return XidCodec.DecodeBinary(Me, data)
    End Function

    Private Sub FinalizeLayout()
        Dim variantBits = If(MaxVariants <= 1, 1, BitLen(MaxVariants - 1))
        Dim countBits = If(MaxObjects <= 1, 1, BitLen(MaxObjects))
        Dim total = CheckBits + countBits + variantBits
        If total > 16 Then Throw New ArgumentException($"header layout exceeds 16 bits: {total}")
        Me.CountBits = countBits
        Me.VariantBits = variantBits
        CheckMask = (1 << CheckBits) - 1
        CountMask = ((1 << countBits) - 1) << CheckBits
        VariantMask = ((1 << variantBits) - 1) << (CheckBits + countBits)
        CountShift = CheckBits
        VariantShift = CheckBits + countBits
    End Sub

    Private Shared Function BitLen(n As Integer) As Integer
        If n <= 0 Then Return 1
        Dim bits = 0
        While n > 0
            n >>= 1
            bits += 1
        End While
        Return bits
    End Function
End Class

End Namespace
