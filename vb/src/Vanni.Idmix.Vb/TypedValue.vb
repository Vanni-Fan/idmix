Namespace Vanni.Idmix

''' <summary>原始类型索引与带类型整数值。</summary> '''
Public NotInheritable Class TypedValue
    Public Const OTypeUint8 As Integer = 0
    Public Const OTypeUint16 As Integer = 1
    Public Const OTypeUint32 As Integer = 2
    Public Const OTypeUint64 As Integer = 3
    Public Const OTypeInt8 As Integer = 4
    Public Const OTypeInt16 As Integer = 5
    Public Const OTypeInt32 As Integer = 6
    Public Const OTypeInt64 As Integer = 7

    Public ReadOnly Property OType As Integer
    Public ReadOnly Property Val As Long

    Public Sub New(otype As Integer, val As Long)
        Me.OType = otype
        Me.Val = val
    End Sub

    Public Shared Function U8(v As Integer) As TypedValue
        Return New TypedValue(OTypeUint8, v)
    End Function

    Public Shared Function U16(v As Integer) As TypedValue
        Return New TypedValue(OTypeUint16, v)
    End Function

    Public Shared Function U32(v As Long) As TypedValue
        Return New TypedValue(OTypeUint32, v)
    End Function

    Public Shared Function U64(v As Long) As TypedValue
        If v < 0 Then Throw New ArgumentOutOfRangeException(NameOf(v), "uint64 overflows")
        Return New TypedValue(OTypeUint64, v)
    End Function

    Public Shared Function I8(v As Integer) As TypedValue
        Return New TypedValue(OTypeInt8, v)
    End Function

    Public Shared Function I16(v As Integer) As TypedValue
        Return New TypedValue(OTypeInt16, v)
    End Function

    Public Shared Function I32(v As Integer) As TypedValue
        Return New TypedValue(OTypeInt32, v)
    End Function

    Public Shared Function I64(v As Long) As TypedValue
        Return New TypedValue(OTypeInt64, v)
    End Function

    Public Overrides Function Equals(obj As Object) As Boolean
        Dim tv = TryCast(obj, TypedValue)
        Return tv IsNot Nothing AndAlso OType = tv.OType AndAlso Val = tv.Val
    End Function

    Public Overrides Function GetHashCode() As Integer
        Return OType * 31 Xor Val.GetHashCode()
    End Function

    Public Overrides Function ToString() As String
        Return $"TypedValue{{otype={OType}, val={Val}}}"
    End Function
End Class

End Namespace
